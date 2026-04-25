import React from 'react';
import { Moon, Sun, Sword, Ghost, Coffee, Zap } from 'lucide-react';
import { Mood } from '../types';

interface MoodBarProps {
  selectedMood: Mood | null;
  onMoodSelect: (mood: Mood | null) => void;
}

const MOODS: { name: Mood; icon: React.ReactNode; color: string }[] = [
  { name: 'Dark', icon: <Moon className="w-4 h-4" />, color: 'hover:bg-purple-500/20 text-purple-400' },
  { name: 'Light', icon: <Sun className="w-4 h-4" />, color: 'hover:bg-yellow-500/20 text-yellow-400' },
  { name: 'Epic', icon: <Sword className="w-4 h-4" />, color: 'hover:bg-red-500/20 text-red-400' },
  { name: 'Ironic', icon: <Zap className="w-4 h-4" />, color: 'hover:bg-blue-500/20 text-blue-400' },
  { name: 'Melancholic', icon: <Ghost className="w-4 h-4" />, color: 'hover:bg-indigo-500/20 text-indigo-400' },
  { name: 'Cozy', icon: <Coffee className="w-4 h-4" />, color: 'hover:bg-orange-500/20 text-orange-400' },
];

export const MoodBar: React.FC<MoodBarProps> = ({ selectedMood, onMoodSelect }) => {
  return (
    <div className="flex items-center gap-2 overflow-x-auto pb-4 no-scrollbar">
      <button
        onClick={() => onMoodSelect(null)}
        className={`px-4 py-2 rounded-full text-sm font-medium transition-all whitespace-nowrap ${
          selectedMood === null
            ? 'bg-brand-500 text-white shadow-lg shadow-brand-500/20'
            : 'bg-white/5 text-slate-400 hover:bg-white/10'
        }`}
      >
        All Vibes
      </button>
      {MOODS.map((mood) => (
        <button
          key={mood.name}
          onClick={() => onMoodSelect(mood.name)}
          className={`flex items-center gap-2 px-4 py-2 rounded-full text-sm font-medium transition-all whitespace-nowrap ${
            selectedMood === mood.name
              ? 'bg-white/20 text-white shadow-lg'
              : `bg-white/5 text-slate-400 ${mood.color}`
          }`}
        >
          {mood.icon}
          {mood.name}
        </button>
      ))}
    </div>
  );
};
