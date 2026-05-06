import { useState, useEffect } from 'react';
import { motion } from 'motion/react';
import { BarChart2, Star, Heart, CheckCircle, Gamepad2, Book, Film } from 'lucide-react';
import { apiGetStats, ApiUserStats } from '../services/api';

const MediaIcon = ({ type }: { type: string }) => {
  switch (type) {
    case 'game':   return <Gamepad2 className="w-5 h-5" />;
    case 'book':   return <Book className="w-5 h-5" />;
    case 'cinema': return <Film className="w-5 h-5" />;
    default:       return <BarChart2 className="w-5 h-5" />;
  }
};

const mediaLabel: Record<string, string> = {
  book:   'Books',
  cinema: 'Movies',
  game:   'Games',
};

function StatCard({
  icon,
  label,
  value,
  sub,
  delay = 0,
}: {
  icon: React.ReactNode;
  label: string;
  value: string | number;
  sub?: string;
  delay?: number;
}) {
  return (
    <motion.div
      initial={{ opacity: 0, y: 24 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay }}
      className="glass-panel rounded-2xl p-6 flex flex-col gap-4"
    >
      <div className="flex items-center gap-3 text-slate-400">
        <div className="w-9 h-9 rounded-xl bg-brand-500/10 flex items-center justify-center text-brand-400">
          {icon}
        </div>
        <span className="text-sm font-semibold uppercase tracking-wider">{label}</span>
      </div>
      <div>
        <p className="font-display font-bold text-4xl text-white">{value}</p>
        {sub && <p className="text-xs text-slate-500 mt-1">{sub}</p>}
      </div>
    </motion.div>
  );
}

export default function Stats() {
  const [stats, setStats] = useState<ApiUserStats | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(false);

  useEffect(() => {
    apiGetStats()
      .then(setStats)
      .catch(() => setError(true))
      .finally(() => setLoading(false));
  }, []);

  if (loading) {
    return (
      <div className="flex items-center justify-center py-32">
        <div className="w-12 h-12 border-4 border-brand-500/20 border-t-brand-500 rounded-full animate-spin" />
      </div>
    );
  }

  if (error || !stats) {
    return (
      <div className="text-center py-32 text-slate-500">
        <BarChart2 className="w-12 h-12 mx-auto mb-4 opacity-30" />
        <p className="text-lg font-medium">Could not load stats</p>
        <p className="text-sm mt-1">Try again later</p>
      </div>
    );
  }

  const isEmpty = stats.total_interactions === 0;

  return (
    <div className="space-y-10 max-w-3xl mx-auto">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
      >
        <h2 className="font-display font-bold text-3xl md:text-4xl mb-1">Your Stats</h2>
        <p className="text-slate-400 text-sm">A snapshot of everything you've tracked in Aura</p>
      </motion.div>

      {isEmpty ? (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="glass-panel rounded-2xl p-12 text-center"
        >
          <BarChart2 className="w-14 h-14 mx-auto mb-4 text-slate-600" />
          <p className="text-xl font-bold text-slate-300">Nothing tracked yet</p>
          <p className="text-slate-500 text-sm mt-2">
            Add items to your library and rate them to see insights here.
          </p>
        </motion.div>
      ) : (
        <>
          {/* Top 3 summary cards */}
          <div className="grid grid-cols-1 sm:grid-cols-3 gap-4">
            <StatCard
              icon={<BarChart2 className="w-5 h-5" />}
              label="Total Tracked"
              value={stats.total_interactions}
              sub={`${stats.completed_count} completed`}
              delay={0}
            />
            <StatCard
              icon={<Star className="w-5 h-5" />}
              label="Avg Rating"
              value={stats.avg_rating > 0 ? stats.avg_rating.toFixed(1) : '—'}
              sub={`${stats.rated_count} rated`}
              delay={0.05}
            />
            <StatCard
              icon={<Heart className="w-5 h-5" />}
              label="Favourites"
              value={stats.favorite_count}
              sub="items marked ♥"
              delay={0.1}
            />
          </div>

          {/* Per-media-type breakdown */}
          {stats.by_media_type.length > 0 && (
            <motion.div
              initial={{ opacity: 0, y: 24 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.15 }}
              className="glass-panel rounded-2xl p-6 space-y-5"
            >
              <h3 className="font-display font-bold text-lg text-white">Breakdown by type</h3>
              <div className="space-y-4">
                {stats.by_media_type.map((row, i) => {
                  const pct = stats.total_interactions > 0
                    ? Math.round((row.total / stats.total_interactions) * 100)
                    : 0;
                  return (
                    <motion.div
                      key={row.media_type}
                      initial={{ opacity: 0, x: -12 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: 0.18 + i * 0.05 }}
                      className="space-y-1.5"
                    >
                      <div className="flex items-center justify-between text-sm">
                        <div className="flex items-center gap-2 text-slate-300 font-medium">
                          <MediaIcon type={row.media_type} />
                          {mediaLabel[row.media_type] ?? row.media_type}
                        </div>
                        <div className="flex items-center gap-4 text-slate-500 text-xs">
                          <span className="flex items-center gap-1">
                            <CheckCircle className="w-3 h-3" />
                            {row.completed}
                          </span>
                          <span className="flex items-center gap-1">
                            <Star className="w-3 h-3" />
                            {row.rated > 0 ? row.avg_rating.toFixed(1) : '—'}
                          </span>
                          <span className="font-bold text-slate-300">{row.total}</span>
                        </div>
                      </div>
                      <div className="w-full h-1.5 bg-white/5 rounded-full overflow-hidden">
                        <motion.div
                          initial={{ width: 0 }}
                          animate={{ width: `${pct}%` }}
                          transition={{ duration: 0.6, delay: 0.2 + i * 0.05, ease: 'easeOut' }}
                          className="h-full bg-brand-500 rounded-full"
                        />
                      </div>
                    </motion.div>
                  );
                })}
              </div>
            </motion.div>
          )}
        </>
      )}
    </div>
  );
}
