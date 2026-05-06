import React from 'react';
import { motion } from 'motion/react';
import { Link } from 'react-router-dom';
import { Gamepad2, Book, Film, Star } from 'lucide-react';
import { MediaItem } from '../types';

interface ContentCardProps {
  item: MediaItem;
}

const TypeIcon = ({ type }: { type: MediaItem['type'] }) => {
  switch (type) {
    case 'game': return <Gamepad2 className="w-4 h-4" />;
    case 'book': return <Book className="w-4 h-4" />;
    case 'movie': return <Film className="w-4 h-4" />;
  }
};

export const ContentCard: React.FC<ContentCardProps> = ({ item }) => {
  const [imgError, setImgError] = React.useState(false);
  const fallback = `https://picsum.photos/seed/${encodeURIComponent(item.id)}/400/600`;

  return (
    <Link to={`/content/${item.id}`}>
      <motion.div
        layout
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        whileHover={{ y: -8 }}
        className="group relative flex flex-col glass-panel rounded-2xl overflow-hidden transition-all hover:shadow-2xl hover:shadow-brand-500/10 h-full"
      >
        <div className="relative aspect-[2/3] overflow-hidden bg-slate-800">
          <img
            src={imgError ? fallback : item.image}
            alt={item.title}
            className="w-full h-full object-cover transition-transform duration-500 group-hover:scale-110"
            referrerPolicy="no-referrer"
            onError={() => setImgError(true)}
            loading="lazy"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-slate-950 via-transparent to-transparent opacity-60" />
          
          <div className="absolute top-3 left-3 px-2 py-1 rounded-lg bg-black/40 backdrop-blur-md border border-white/10 flex items-center gap-1.5 text-[10px] font-bold uppercase tracking-wider text-white">
            <TypeIcon type={item.type} />
            {item.type}
          </div>

          {item.rating > 0 && (
            <div className="absolute top-3 right-3 px-2 py-1 rounded-lg bg-black/40 backdrop-blur-md border border-white/10 flex items-center gap-1 text-[10px] font-bold text-yellow-400">
              <Star className="w-3 h-3 fill-yellow-400" />
              {item.rating.toFixed(1)}
            </div>
          )}
        </div>

        <div className="p-4 flex flex-col gap-3">
          <div>
            <h3 className="font-display font-bold text-base leading-tight group-hover:text-brand-500 transition-colors line-clamp-2">
              {item.title}
            </h3>
            <p className="text-xs text-slate-400 mt-1 line-clamp-1">
              {[item.setting, item.genre.join(', ')].filter(Boolean).join(' • ')}
            </p>
          </div>

          <div className="flex flex-wrap gap-1.5 min-h-[20px]">
            {item.tonality && (
              <span className="px-2 py-0.5 rounded-md bg-brand-500/10 text-brand-400 text-[10px] font-semibold border border-brand-500/20">
                {item.tonality}
              </span>
            )}
            {item.themes.slice(0, 2).map((theme) => (
              <span key={theme} className="px-2 py-0.5 rounded-md bg-white/5 text-slate-300 text-[10px] font-medium border border-white/10">
                {theme}
              </span>
            ))}
            {!item.tonality && item.themes.length === 0 && item.genre.length > 0 && (
              <span className="px-2 py-0.5 rounded-md bg-white/5 text-slate-400 text-[10px] font-medium border border-white/10">
                {item.genre[0]}
              </span>
            )}
          </div>

          {item.matchReason && (
            <div className="mt-2 pt-2 border-t border-white/5">
              <p className="text-[10px] text-slate-500 italic leading-relaxed">
                "{item.matchReason}"
              </p>
            </div>
          )}
        </div>
      </motion.div>
    </Link>
  );
};
