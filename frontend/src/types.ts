export type MediaType = 'game' | 'book' | 'movie';

export interface MediaItem {
  id: string;
  type: MediaType;
  title: string;
  image: string;
  // Common Criteria
  genre: string[];
  setting: string;
  themes: string[];
  tonality: string;
  targetAudience: string;
  // Specifics
  platform?: string[];
  volume?: string; // e.g. "120 min", "350 pages", "20 hours"
  format?: string;
  rating: number;
  matchReason?: string;
  matchScore?: number;
}

export type Mood = 'Dark' | 'Light' | 'Epic' | 'Ironic' | 'Melancholic' | 'Cozy';

export interface MoodConfig {
  name: Mood;
  color: string;
  icon: string;
}
