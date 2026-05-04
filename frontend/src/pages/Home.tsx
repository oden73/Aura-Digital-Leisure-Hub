import { useState, useMemo, useEffect, useRef } from 'react';
import { motion, AnimatePresence } from 'motion/react';
import { MoodBar } from '../components/MoodBar';
import { SearchBar } from '../components/SearchBar';
import { ContentCard } from '../components/ContentCard';
import { MediaItem, Mood } from '../types';
import { apiSearch, mapApiItem } from '../services/api';

export default function Home() {
  const [selectedMood, setSelectedMood] = useState<Mood | null>(null);
  const [searchQuery, setSearchQuery] = useState('');
  const [allItems, setAllItems] = useState<MediaItem[]>([]);
  const [filteredItems, setFilteredItems] = useState<MediaItem[]>([]);
  const [isSearching, setIsSearching] = useState(false);
  const abortRef = useRef<AbortController | null>(null);

  useEffect(() => {
    apiSearch('', 40)
      .then(data => {
        const mapped = data.map(mapApiItem);
        setAllItems(mapped);
        setFilteredItems(mapped);
      })
      .catch(() => {});
  }, []);

  useEffect(() => {
    if (abortRef.current) abortRef.current.abort();

    if (!searchQuery) {
      const filtered = selectedMood
        ? allItems.filter(i => i.tonality === selectedMood)
        : allItems;
      setFilteredItems(filtered);
      return;
    }

    const controller = new AbortController();
    abortRef.current = controller;
    setIsSearching(true);

    apiSearch(searchQuery, 20)
      .then(data => {
        if (controller.signal.aborted) return;
        let mapped = data.map(mapApiItem);
        if (selectedMood) mapped = mapped.filter(i => i.tonality === selectedMood);
        setFilteredItems(mapped);
      })
      .catch(() => {})
      .finally(() => {
        if (!controller.signal.aborted) setIsSearching(false);
      });
  }, [selectedMood, searchQuery, allItems]);

  const thematicRows = useMemo(() => {
    const themes = Array.from(new Set(filteredItems.flatMap(i => i.themes))).slice(0, 4);
    return themes
      .map(theme => ({
        title: `Exploring: ${theme}`,
        items: filteredItems.filter(i => i.themes.includes(theme)),
      }))
      .filter(row => row.items.length > 0);
  }, [filteredItems]);

  return (
    <div className="space-y-12">
      <div className="text-center mb-12">
        <motion.h2
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          className="font-display font-bold text-4xl md:text-6xl mb-6 tracking-tight"
        >
          Your Personal <span className="text-brand-500">Leisure Hub</span>
        </motion.h2>
        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
          className="text-slate-400 max-w-xl mx-auto text-lg mb-10"
        >
          Stop searching, start experiencing. Aura bridges games, books, and movies through shared DNA.
        </motion.p>

        <SearchBar onSearch={setSearchQuery} />
      </div>

      <MoodBar selectedMood={selectedMood} onMoodSelect={setSelectedMood} />

      <main className="space-y-16">
        {isSearching ? (
          <div className="flex flex-col items-center justify-center py-20 gap-4">
            <div className="w-12 h-12 border-4 border-brand-500/20 border-t-brand-500 rounded-full animate-spin" />
            <p className="text-slate-400 font-medium animate-pulse">AI is curating your perfect mix...</p>
          </div>
        ) : (
          <AnimatePresence mode="popLayout">
            {searchQuery ? (
              <section key="search-results">
                <div className="flex items-center justify-between mb-8">
                  <h3 className="font-display font-bold text-2xl">Search Results</h3>
                  <span className="text-sm text-slate-500">{filteredItems.length} matches found</span>
                </div>
                <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
                  {filteredItems.map((item) => (
                    <ContentCard key={item.id} item={item} />
                  ))}
                </div>
              </section>
            ) : (
              <div key="discovery-feed" className="space-y-16">
                <section>
                  <div className="flex items-center justify-between mb-8">
                    <h3 className="font-display font-bold text-2xl">Recommended for You</h3>
                  </div>
                  <div className="grid grid-cols-2 md:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-6">
                    {filteredItems.slice(0, 5).map((item) => (
                      <ContentCard key={item.id} item={item} />
                    ))}
                  </div>
                </section>

                {thematicRows.map((row) => (
                  <section key={row.title}>
                    <div className="flex items-center justify-between mb-8">
                      <h3 className="font-display font-bold text-2xl">{row.title}</h3>
                    </div>
                    <div className="flex gap-6 overflow-x-auto pb-4 no-scrollbar snap-x">
                      {row.items.map((item) => (
                        <div key={item.id} className="w-64 flex-shrink-0 snap-start">
                          <ContentCard item={item} />
                        </div>
                      ))}
                    </div>
                  </section>
                ))}
              </div>
            )}
          </AnimatePresence>
        )}

        {!isSearching && filteredItems.length === 0 && (
          <div className="text-center py-20">
            <p className="text-slate-500 text-lg">No matches found for this vibe. Try exploring something else!</p>
            <button
              onClick={() => { setSelectedMood(null); setSearchQuery(''); }}
              className="mt-4 text-brand-500 font-semibold hover:underline"
            >
              Reset filters
            </button>
          </div>
        )}
      </main>
    </div>
  );
}
