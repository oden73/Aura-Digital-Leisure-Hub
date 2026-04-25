import type { FC } from 'react';
import { Sparkles } from 'lucide-react';

interface LoadingScreenProps {
  /** Optional caption shown under the spinner. Defaults to a generic label. */
  message?: string;
}

/**
 * Full-viewport loading state used while the app is bootstrapping
 * (Firebase auth resolution, lazy-loaded chunks, etc.). Matches the
 * dark/glass aesthetic of the rest of the app so it doesn't read as a
 * crash or a blank tab.
 */
export const LoadingScreen: FC<LoadingScreenProps> = ({
  message = 'Tuning your Aura…',
}) => (
  <div className="min-h-screen bg-slate-950 text-slate-200 flex flex-col items-center justify-center gap-6 px-6">
    <div className="relative">
      <div className="w-16 h-16 rounded-2xl bg-brand-500/20 flex items-center justify-center shadow-lg shadow-brand-500/10">
        <Sparkles className="w-8 h-8 text-brand-500 animate-pulse" />
      </div>
      <div className="absolute -inset-2 rounded-3xl border-2 border-brand-500/30 border-t-brand-500 animate-spin" />
    </div>
    <p className="text-slate-400 font-medium tracking-wide">{message}</p>
  </div>
);
