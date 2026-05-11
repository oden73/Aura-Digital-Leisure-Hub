import React, { useState } from 'react';
import { motion, AnimatePresence } from 'motion/react';
import {
  Sparkles, Gamepad2, ChevronRight, Check, X, Loader2,
} from 'lucide-react';
import { toast } from 'sonner';
import { apiLinkExternalAccount } from '../services/api';
import { useAuth } from '../contexts/AuthContext';

const GENRES = [
  'Action', 'Adventure', 'RPG', 'Strategy', 'Horror',
  'Sci-Fi', 'Fantasy', 'Casual', 'Puzzle', 'Simulation',
  'Sports', 'Fighting', 'Platformer', 'Sandbox', 'Stealth',
];

interface WelcomeModalProps {
  username: string;
}

export const WelcomeModal: React.FC<WelcomeModalProps> = ({ username }) => {
  const { dismissWelcomeModal } = useAuth();
  const [step, setStep] = useState<1 | 2>(1);
  const [steamId, setSteamId] = useState('');
  const [linking, setLinking] = useState(false);
  const [selectedGenres, setSelectedGenres] = useState<Set<string>>(new Set());

  const handleLinkSteam = async () => {
    const id = steamId.trim();
    if (!id) { setStep(2); return; }
    setLinking(true);
    try {
      await apiLinkExternalAccount('steam', id);
      toast.success('Steam account linked!');
    } catch {
      toast.error('Could not link Steam — you can connect it later in settings.');
    } finally {
      setLinking(false);
      setStep(2);
    }
  };

  const toggleGenre = (g: string) => {
    setSelectedGenres(prev => {
      const next = new Set(prev);
      if (next.has(g)) next.delete(g); else next.add(g);
      return next;
    });
  };

  const handleFinish = () => {
    if (selectedGenres.size > 0) {
      localStorage.setItem('aura_preferred_genres', JSON.stringify([...selectedGenres]));
    }
    dismissWelcomeModal();
    toast.success("You're all set! Enjoy Aura.");
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm">
      <motion.div
        initial={{ opacity: 0, scale: 0.92, y: 20 }}
        animate={{ opacity: 1, scale: 1, y: 0 }}
        exit={{ opacity: 0, scale: 0.92, y: 20 }}
        className="relative w-full max-w-lg glass-panel rounded-3xl p-8 shadow-2xl border border-white/10"
      >
        <button
          onClick={dismissWelcomeModal}
          className="absolute top-4 right-4 p-2 rounded-xl text-slate-500 hover:text-slate-300 hover:bg-white/5 transition-colors"
        >
          <X className="w-5 h-5" />
        </button>

        {/* Progress dots */}
        <div className="flex justify-center gap-2 mb-8">
          {[1, 2].map(n => (
            <div
              key={n}
              className={`h-1.5 rounded-full transition-all duration-300 ${
                n === step ? 'w-8 bg-brand-500' : n < step ? 'w-4 bg-brand-500/40' : 'w-4 bg-white/10'
              }`}
            />
          ))}
        </div>

        <AnimatePresence mode="wait">
          {step === 1 ? (
            <motion.div
              key="step1"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              className="space-y-6"
            >
              <div className="text-center">
                <div className="w-14 h-14 rounded-2xl bg-brand-500 flex items-center justify-center shadow-lg shadow-brand-500/20 mx-auto mb-4">
                  <Sparkles className="w-8 h-8 text-white" />
                </div>
                <h2 className="font-display font-bold text-2xl">
                  Welcome, {username}!
                </h2>
                <p className="text-slate-400 mt-2 text-sm">
                  Let's personalise your experience. Connect your services to unlock smart recommendations.
                </p>
              </div>

              <div className="space-y-3">
                <p className="text-xs font-semibold text-slate-500 uppercase tracking-widest">
                  Connect a service
                </p>
                <div className="glass-panel rounded-2xl p-4 border border-white/10 space-y-3">
                  <div className="flex items-center gap-3">
                    <div className="w-9 h-9 rounded-xl bg-[#1b2838] flex items-center justify-center flex-shrink-0">
                      <Gamepad2 className="w-5 h-5 text-[#66c0f4]" />
                    </div>
                    <div>
                      <p className="font-semibold text-sm">Steam</p>
                      <p className="text-xs text-slate-500">Import your game library</p>
                    </div>
                  </div>
                  <input
                    type="text"
                    value={steamId}
                    onChange={e => setSteamId(e.target.value)}
                    placeholder="Steam ID (e.g. 76561198000000000)"
                    className="w-full bg-white/5 border border-white/10 rounded-xl py-2.5 px-4 text-sm text-slate-200 placeholder-slate-500 focus:outline-none focus:ring-2 focus:ring-brand-500/50 focus:bg-white/10 transition-all"
                  />
                  <p className="text-[11px] text-slate-600">
                    Find your Steam ID at{' '}
                    <a
                      href="https://www.steamidfinder.com"
                      target="_blank"
                      rel="noreferrer"
                      className="text-brand-500 hover:underline"
                    >
                      steamidfinder.com
                    </a>
                  </p>
                </div>
              </div>

              <div className="flex gap-3">
                <button
                  onClick={dismissWelcomeModal}
                  className="flex-1 py-3 rounded-2xl border border-white/10 text-slate-400 text-sm font-medium hover:bg-white/5 transition-colors"
                >
                  Skip for now
                </button>
                <button
                  onClick={handleLinkSteam}
                  disabled={linking}
                  className="flex-1 py-3 rounded-2xl bg-brand-500 hover:bg-brand-600 text-white text-sm font-bold shadow-lg shadow-brand-500/20 transition-all flex items-center justify-center gap-2 disabled:opacity-60"
                >
                  {linking ? (
                    <Loader2 className="w-4 h-4 animate-spin" />
                  ) : (
                    <>
                      Next
                      <ChevronRight className="w-4 h-4" />
                    </>
                  )}
                </button>
              </div>
            </motion.div>
          ) : (
            <motion.div
              key="step2"
              initial={{ opacity: 0, x: 20 }}
              animate={{ opacity: 1, x: 0 }}
              exit={{ opacity: 0, x: -20 }}
              className="space-y-6"
            >
              <div className="text-center">
                <h2 className="font-display font-bold text-2xl">
                  What do you love?
                </h2>
                <p className="text-slate-400 mt-2 text-sm">
                  Pick genres you enjoy — we'll use these to fine-tune your recommendations.
                </p>
              </div>

              <div className="flex flex-wrap gap-2 justify-center">
                {GENRES.map(g => {
                  const active = selectedGenres.has(g);
                  return (
                    <button
                      key={g}
                      onClick={() => toggleGenre(g)}
                      className={`flex items-center gap-1.5 px-3.5 py-2 rounded-xl text-sm font-medium border transition-all ${
                        active
                          ? 'bg-brand-500/20 border-brand-500/50 text-brand-400'
                          : 'bg-white/5 border-white/10 text-slate-400 hover:bg-white/10 hover:text-slate-200'
                      }`}
                    >
                      {active && <Check className="w-3.5 h-3.5" />}
                      {g}
                    </button>
                  );
                })}
              </div>

              <div className="flex gap-3">
                <button
                  onClick={() => setStep(1)}
                  className="flex-1 py-3 rounded-2xl border border-white/10 text-slate-400 text-sm font-medium hover:bg-white/5 transition-colors"
                >
                  Back
                </button>
                <button
                  onClick={handleFinish}
                  className="flex-1 py-3 rounded-2xl bg-brand-500 hover:bg-brand-600 text-white text-sm font-bold shadow-lg shadow-brand-500/20 transition-all flex items-center justify-center gap-2"
                >
                  <Check className="w-4 h-4" />
                  {selectedGenres.size > 0 ? "Let's go!" : 'Skip genres'}
                </button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </motion.div>
    </div>
  );
};
