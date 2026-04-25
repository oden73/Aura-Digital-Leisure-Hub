import { GoogleGenAI, Type } from "@google/genai";
import { MediaItem } from "../types";

const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY || '' });

export async function processAIQuery(query: string, items: MediaItem[]): Promise<MediaItem[]> {
  if (!query.trim()) return items;

  try {
    const response = await ai.models.generateContent({
      model: "gemini-3-flash-preview",
      contents: `
        You are an expert content curator for a cross-media hub. 
        Given a user query and a list of media items (games, books, movies), filter and rank the items based on how well they match the query.
        
        User Query: "${query}"
        
        Media Items:
        ${JSON.stringify(items.map(i => ({ id: i.id, title: i.title, type: i.type, tonality: i.tonality, themes: i.themes, setting: i.setting })), null, 2)}
      `,
      config: {
        responseMimeType: "application/json",
        responseSchema: {
          type: Type.ARRAY,
          items: {
            type: Type.OBJECT,
            properties: {
              id: { type: Type.STRING },
              matchReason: { type: Type.STRING, description: "A short, catchy explanation (max 15 words) of why this matches the query." },
              matchScore: { type: Type.NUMBER, description: "A number from 0 to 1." }
            },
            required: ["id", "matchReason", "matchScore"]
          }
        }
      }
    });

    const text = response.text;
    if (!text) return items;
    
    const aiResults = JSON.parse(text);

    return items
      .map(item => {
        const aiMatch = aiResults.find((r: any) => r.id === item.id);
        if (aiMatch && aiMatch.matchScore > 0.3) {
          return { ...item, matchReason: aiMatch.matchReason, matchScore: aiMatch.matchScore } as MediaItem;
        }
        return null;
      })
      .filter((i): i is MediaItem => i !== null)
      .sort((a, b) => (b.matchScore || 0) - (a.matchScore || 0));

  } catch (error) {
    console.error("AI Query Error:", error);
    // Fallback to simple keyword matching if AI fails
    return items.filter(item => 
      item.title.toLowerCase().includes(query.toLowerCase()) ||
      item.themes.some(t => t.toLowerCase().includes(query.toLowerCase())) ||
      item.tonality.toLowerCase().includes(query.toLowerCase())
    );
  }
}
