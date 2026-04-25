import React, { useState, useEffect } from 'react';
import { Search, Sparkles } from 'lucide-react';

interface SearchBarProps {
  onSearch: (query: string) => void;
}

const PLACEHOLDERS = [
  "Something dark about growth in a fantasy world...",
  "Epic sci-fi with political intrigue...",
  "Cozy mystery for a rainy evening...",
  "Ironic take on corporate power...",
  "Melancholic journey through space..."
];

export const SearchBar: React.FC<SearchBarProps> = ({ onSearch }) => {
  const [query, setQuery] = useState('');
  const [placeholderIndex, setPlaceholderIndex] = useState(0);

  useEffect(() => {
    const interval = setInterval(() => {
      setPlaceholderIndex((prev) => (prev + 1) % PLACEHOLDERS.length);
    }, 4000);
    return () => clearInterval(interval);
  }, []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    onSearch(query);
  };

  return (
    <form onSubmit={handleSubmit} className="relative group max-w-2xl mx-auto w-full">
      <div className="absolute inset-y-0 left-4 flex items-center pointer-events-none">
        <Search className="w-5 h-5 text-slate-500 group-focus-within:text-brand-500 transition-colors" />
      </div>
      <input
        type="text"
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={PLACEHOLDERS[placeholderIndex]}
        className="w-full bg-white/5 border border-white/10 rounded-2xl py-4 pl-12 pr-12 text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-brand-500/50 focus:bg-white/10 transition-all font-medium"
      />
      <div className="absolute inset-y-0 right-4 flex items-center">
        <button
          type="submit"
          className="p-2 rounded-xl bg-brand-500/10 text-brand-400 hover:bg-brand-500 hover:text-white transition-all"
        >
          <Sparkles className="w-4 h-4" />
        </button>
      </div>
    </form>
  );
};
