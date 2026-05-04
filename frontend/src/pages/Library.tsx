import React, { useState, useEffect } from 'react';
import { motion, AnimatePresence } from 'motion/react';
import { useNavigate } from 'react-router-dom';
import { useAuth } from '../contexts/AuthContext';
import { apiGetLibraryItems, mapApiItem } from '../services/api';
import { ContentCard } from '../components/ContentCard';
import { MediaItem } from '../types';
import { Library, LayoutGrid, List as ListIcon, Search } from 'lucide-react';

export default function LibraryPage() {
  const { user } = useAuth();
  const navigate = useNavigate();
  const [displayItems, setDisplayItems] = useState<MediaItem[]>([]);
  const [loading, setLoading] = useState(true);
  const [filter, setFilter] = useState<'all' | 'game' | 'book' | 'movie'>('all');
  const [viewMode, setViewMode] = useState<'grid' | 'list'>('grid');

  useEffect(() => {
    if (!user) { setLoading(false); return; }

    apiGetLibraryItems(100)
      .then(items => {
        setDisplayItems(items.map(li => mapApiItem(li.item)));
      })
      .catch(() => {})
      .finally(() => setLoading(false));
  }, [user]);

  const filteredItems = displayItems.filter(item =>
    filter === 'all' ? true : item.type === filter,
  );

  if (!user) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center text-center px-6">
        <div className="w-20 h-20 rounded-3xl bg-brand-500/10 flex items-center justify-center mb-6">
          <Library className="w-10 h-10 text-brand-500" />
        </div>
        <h2 className="text-3xl font-bold mb-4">Your Library is Waiting</h2>
        <p className="text-slate-400 max-w-md mb-8">
          Sign in to start building your personal collection of games, books, and movies.
        </p>
        <button
          onClick={() => navigate('/login')}
          className="bg-brand-500 hover:bg-brand-600 text-white font-bold px-8 py-4 rounded-2xl shadow-lg shadow-brand-500/20 transition-all"
        >
          Sign In Now
        </button>
      </div>
    );
  }

  return (
    <div className="space-y-8 pb-12">
      <header className="flex flex-col md:flex-row md:items-end justify-between gap-6">
        <div>
          <h1 className="font-display font-bold text-4xl mb-2">My Library</h1>
          <p className="text-slate-400">Manage your curated collection of digital experiences.</p>
        </div>

        <div className="flex items-center gap-3">
          <div className="flex bg-white/5 p-1 rounded-xl border border-white/10">
            <button
              onClick={() => setViewMode('grid')}
              className={`p-2 rounded-lg transition-all ${viewMode === 'grid' ? 'bg-brand-500 text-white shadow-lg' : 'text-slate-400 hover:text-slate-200'}`}
            >
              <LayoutGrid className="w-5 h-5" />
            </button>
            <button
              onClick={() => setViewMode('list')}
              className={`p-2 rounded-lg transition-all ${viewMode === 'list' ? 'bg-brand-500 text-white shadow-lg' : 'text-slate-400 hover:text-slate-200'}`}
            >
              <ListIcon className="w-5 h-5" />
            </button>
          </div>

          <select
            value={filter}
            onChange={(e) => setFilter(e.target.value as typeof filter)}
            className="bg-white/5 border border-white/10 rounded-xl px-4 py-2.5 text-sm font-medium focus:outline-none focus:ring-2 focus:ring-brand-500/50"
          >
            <option value="all">All Media</option>
            <option value="game">Games</option>
            <option value="book">Books</option>
            <option value="movie">Movies</option>
          </select>
        </div>
      </header>

      {loading ? (
        <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
          {[...Array(5)].map((_, i) => (
            <div key={i} className="aspect-[2/3] rounded-2xl bg-white/5 animate-pulse" />
          ))}
        </div>
      ) : filteredItems.length > 0 ? (
        <motion.div
          layout
          className={viewMode === 'grid'
            ? 'grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6'
            : 'space-y-4'
          }
        >
          <AnimatePresence mode="popLayout">
            {filteredItems.map((item) =>
              viewMode === 'grid' ? (
                <ContentCard key={item.id} item={item} />
              ) : (
                <motion.div
                  key={item.id}
                  initial={{ opacity: 0, x: -20 }}
                  animate={{ opacity: 1, x: 0 }}
                  exit={{ opacity: 0, scale: 0.95 }}
                  className="glass-panel p-4 rounded-2xl flex items-center gap-6 group hover:border-brand-500/30 transition-colors"
                >
                  <div className="w-16 h-20 rounded-lg overflow-hidden flex-shrink-0">
                    <img src={item.image} alt={item.title} className="w-full h-full object-cover" referrerPolicy="no-referrer" />
                  </div>
                  <div className="flex-grow">
                    <h3 className="font-bold text-lg">{item.title}</h3>
                    <p className="text-xs text-slate-500 uppercase tracking-widest font-bold">{item.type} • {item.tonality}</p>
                  </div>
                  <div className="hidden md:block text-right">
                    <p className="text-xs text-slate-500 mb-1">Status</p>
                    <span className="px-3 py-1 rounded-full bg-brand-500/10 text-brand-400 text-xs font-bold border border-brand-500/20">
                      Planned
                    </span>
                  </div>
                  <button
                    onClick={() => navigate(`/content/${item.id}`)}
                    aria-label={`Open ${item.title}`}
                    className="p-3 rounded-xl bg-white/5 hover:bg-brand-500 hover:text-white transition-all"
                  >
                    <Search className="w-5 h-5" />
                  </button>
                </motion.div>
              ),
            )}
          </AnimatePresence>
        </motion.div>
      ) : (
        <div className="text-center py-20 glass-panel rounded-3xl border-dashed border-white/10">
          <p className="text-slate-500 text-lg mb-4">Your library is empty.</p>
          <button
            onClick={() => navigate('/')}
            className="text-brand-500 font-bold hover:underline"
          >
            Go discover something new
          </button>
        </div>
      )}
    </div>
  );
}
