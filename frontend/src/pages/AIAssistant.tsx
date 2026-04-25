import React, { useState, useRef, useEffect } from 'react';
import { motion } from 'motion/react';
import { useNavigate } from 'react-router-dom';
import { Send, Bot, User, Trash2 } from 'lucide-react';
import { GoogleGenAI } from "@google/genai";
import { MOCK_DATA } from '../data';

const ai = new GoogleGenAI({ apiKey: process.env.GEMINI_API_KEY || '' });

interface Message {
  id: string;
  role: 'user' | 'assistant';
  content: string;
  recommendations?: string[]; // IDs of recommended items
  timestamp: Date;
}

export default function AIAssistant() {
  const navigate = useNavigate();
  const [messages, setMessages] = useState<Message[]>([
    {
      id: '1',
      role: 'assistant',
      content: "Hello! I'm Aura, your personal leisure curator. Unlike the quick search bar, I'm here to dive deep into your tastes. Tell me about a movie or game you loved, and I'll find the 'Shared DNA' in books or other media for you.",
      timestamp: new Date()
    }
  ]);
  const [input, setInput] = useState('');
  const [isTyping, setIsTyping] = useState(false);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const scrollToBottom = () => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" });
  };

  useEffect(() => {
    scrollToBottom();
  }, [messages]);

  const handleSend = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim() || isTyping) return;

    const userMessage: Message = {
      id: Date.now().toString(),
      role: 'user',
      content: input,
      timestamp: new Date()
    };

    setMessages(prev => [...prev, userMessage]);
    setInput('');
    setIsTyping(true);

    try {
      const response = await ai.models.generateContent({
        model: "gemini-3-flash-preview",
        contents: `
          You are Aura, an expert cross-media curator. You help users find connections between games, books, and movies.
          
          Catalog Data:
          ${JSON.stringify(MOCK_DATA.map(i => ({ id: i.id, title: i.title, type: i.type, tonality: i.tonality, themes: i.themes })), null, 2)}
          
          User Query: "${input}"
          
          Respond in JSON format:
          {
            "text": "Your conversational response explaining the connections and why you recommend these items.",
            "recommendationIds": ["list", "of", "matching", "ids", "from", "catalog"]
          }
        `,
        config: {
          responseMimeType: "application/json"
        }
      });

      const result = JSON.parse(response.text || '{}');

      const assistantMessage: Message = {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: result.text || "I found some interesting connections for you.",
        recommendations: result.recommendationIds,
        timestamp: new Date()
      };

      setMessages(prev => [...prev, assistantMessage]);
    } catch (error) {
      console.error("AI Assistant Error:", error);
      setMessages(prev => [...prev, {
        id: (Date.now() + 1).toString(),
        role: 'assistant',
        content: "I'm having a bit of trouble connecting to my creative core. Could you try again in a moment?",
        timestamp: new Date()
      }]);
    } finally {
      setIsTyping(false);
    }
  };

  return (
    <div className="max-w-4xl mx-auto h-[calc(100vh-200px)] flex flex-col glass-panel rounded-3xl overflow-hidden shadow-2xl border-white/10">
      {/* Header */}
      <header className="p-6 border-b border-white/10 bg-white/5 flex items-center justify-between">
        <div className="flex items-center gap-3">
          <div className="w-10 h-10 rounded-xl bg-brand-500 flex items-center justify-center shadow-lg shadow-brand-500/20">
            <Bot className="w-6 h-6 text-white" />
          </div>
          <div>
            <h2 className="font-display font-bold text-xl">Aura Curator</h2>
            <div className="flex items-center gap-1.5">
              <span className="w-2 h-2 rounded-full bg-green-500 animate-pulse" />
              <span className="text-[10px] text-slate-500 uppercase font-bold tracking-widest">Deep Discovery Mode</span>
            </div>
          </div>
        </div>
        <button 
          onClick={() => setMessages([messages[0]])}
          className="p-2 rounded-lg hover:bg-white/5 text-slate-400 transition-colors"
        >
          <Trash2 className="w-5 h-5" />
        </button>
      </header>

      {/* Messages */}
      <div className="flex-grow overflow-y-auto p-6 space-y-8 no-scrollbar">
        {messages.map((msg) => (
          <motion.div
            key={msg.id}
            initial={{ opacity: 0, y: 10 }}
            animate={{ opacity: 1, y: 0 }}
            className={`flex items-start gap-4 ${msg.role === 'user' ? 'flex-row-reverse' : ''}`}
          >
            <div className={`w-8 h-8 rounded-lg flex items-center justify-center flex-shrink-0 ${
              msg.role === 'assistant' ? 'bg-brand-500 text-white' : 'bg-white/10 text-slate-400'
            }`}>
              {msg.role === 'assistant' ? <Bot className="w-5 h-5" /> : <User className="w-5 h-5" />}
            </div>
            <div className="flex flex-col gap-3 max-w-[85%]">
              <div className={`p-4 rounded-2xl text-sm leading-relaxed ${
                msg.role === 'assistant' 
                  ? 'bg-white/5 border border-white/10 text-slate-200 rounded-tl-none' 
                  : 'bg-brand-500 text-white rounded-tr-none shadow-lg shadow-brand-500/20'
              }`}>
                {msg.content}
              </div>

              {/* Recommendations Grid inside Chat */}
              {msg.recommendations && msg.recommendations.length > 0 && (
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-3 mt-2">
                  {msg.recommendations.map(id => {
                    const item = MOCK_DATA.find(m => m.id === id);
                    if (!item) return null;
                    return (
                      <motion.div 
                        key={id}
                        whileHover={{ scale: 1.02 }}
                        onClick={() => navigate(`/content/${id}`)}
                        className="glass-panel p-2 rounded-xl cursor-pointer border-brand-500/20 hover:border-brand-500/50 transition-all"
                      >
                        <div className="aspect-[2/3] rounded-lg overflow-hidden mb-2">
                          <img src={item.image} alt={item.title} className="w-full h-full object-cover" referrerPolicy="no-referrer" />
                        </div>
                        <p className="text-[10px] font-bold truncate text-white">{item.title}</p>
                        <p className="text-[8px] text-slate-500 uppercase tracking-tighter">{item.type}</p>
                      </motion.div>
                    );
                  })}
                </div>
              )}
              
              <div className={`text-[10px] opacity-30 ${msg.role === 'user' ? 'text-right' : ''}`}>
                {msg.timestamp.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })}
              </div>
            </div>
          </motion.div>
        ))}
        {isTyping && (
          <div className="flex items-center gap-4">
            <div className="w-8 h-8 rounded-lg bg-brand-500 text-white flex items-center justify-center">
              <Bot className="w-5 h-5" />
            </div>
            <div className="bg-white/5 border border-white/10 p-4 rounded-2xl rounded-tl-none flex gap-1">
              <span className="w-1.5 h-1.5 rounded-full bg-slate-500 animate-bounce" />
              <span className="w-1.5 h-1.5 rounded-full bg-slate-500 animate-bounce [animation-delay:0.2s]" />
              <span className="w-1.5 h-1.5 rounded-full bg-slate-500 animate-bounce [animation-delay:0.4s]" />
            </div>
          </div>
        )}
        <div ref={messagesEndRef} />
      </div>

      {/* Input */}
      <footer className="p-6 border-t border-white/10 bg-white/5">
        <form onSubmit={handleSend} className="relative group">
          <input 
            type="text" 
            value={input}
            onChange={(e) => setInput(e.target.value)}
            placeholder="Ask Aura for a deep recommendation..."
            className="w-full bg-slate-900/50 border border-white/10 rounded-2xl py-4 pl-6 pr-14 text-slate-200 focus:outline-none focus:ring-2 focus:ring-brand-500/50 focus:bg-slate-900 transition-all"
          />
          <button 
            type="submit"
            disabled={!input.trim() || isTyping}
            className="absolute right-2 top-1/2 -translate-y-1/2 w-10 h-10 rounded-xl bg-brand-500 text-white flex items-center justify-center shadow-lg shadow-brand-500/20 hover:bg-brand-600 disabled:opacity-50 transition-all"
          >
            <Send className="w-5 h-5" />
          </button>
        </form>
      </footer>
    </div>
  );
}
