import React from 'react';
import { Link, useLocation, Outlet, useNavigate } from 'react-router-dom';
import { Sparkles, Info, BarChart2, LogIn, LogOut, X, Book, Film, Gamepad2, Brain } from 'lucide-react';
import { motion, AnimatePresence } from 'motion/react';
import { toast } from 'sonner';
import { useAuth } from '../contexts/AuthContext';

const AboutModal: React.FC<{ onClose: () => void }> = ({ onClose }) => (
  <div
    className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm"
    onClick={onClose}
  >
    <motion.div
      initial={{ opacity: 0, scale: 0.93, y: 16 }}
      animate={{ opacity: 1, scale: 1, y: 0 }}
      exit={{ opacity: 0, scale: 0.93, y: 16 }}
      transition={{ duration: 0.2 }}
      className="relative w-full max-w-md glass-panel rounded-3xl p-8 shadow-2xl border border-white/10"
      onClick={e => e.stopPropagation()}
    >
      <button
        onClick={onClose}
        className="absolute top-4 right-4 p-2 rounded-xl text-slate-500 hover:text-slate-300 hover:bg-white/5 transition-colors"
      >
        <X className="w-5 h-5" />
      </button>

      <div className="flex items-center gap-3 mb-6">
        <div className="w-12 h-12 rounded-2xl bg-brand-500 flex items-center justify-center shadow-lg shadow-brand-500/20">
          <Sparkles className="w-7 h-7 text-white" />
        </div>
        <div>
          <h2 className="font-display font-bold text-xl">About Aura</h2>
          <p className="text-xs text-slate-500">Your personal media compass</p>
        </div>
      </div>

      <p className="text-sm text-slate-300 leading-relaxed mb-6">
        Aura helps you discover and track the media you love — games, movies, and books —
        all in one place. Tell us your mood or taste and our AI will surface what fits you
        perfectly right now.
      </p>

      <div className="grid grid-cols-3 gap-3 mb-6">
        {[
          { icon: Gamepad2, label: 'Games', color: 'text-[#66c0f4]', bg: 'bg-[#1b2838]' },
          { icon: Film, label: 'Movies', color: 'text-rose-400', bg: 'bg-rose-500/10' },
          { icon: Book, label: 'Books', color: 'text-amber-400', bg: 'bg-amber-500/10' },
        ].map(({ icon: Icon, label, color, bg }) => (
          <div key={label} className={`flex flex-col items-center gap-2 p-3 rounded-2xl border border-white/10 ${bg}`}>
            <Icon className={`w-5 h-5 ${color}`} />
            <span className="text-xs font-medium text-slate-300">{label}</span>
          </div>
        ))}
      </div>

      <div className="flex items-start gap-3 p-4 rounded-2xl bg-brand-500/10 border border-brand-500/20">
        <Brain className="w-5 h-5 text-brand-400 flex-shrink-0 mt-0.5" />
        <p className="text-xs text-slate-400 leading-relaxed">
          Connect your Steam account to let the AI learn from your library and deliver
          hyper-personal recommendations.
        </p>
      </div>
    </motion.div>
  </div>
);

export const Layout: React.FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, logout } = useAuth();
  const isAuthPage = ['/login', '/register'].includes(location.pathname);
  const [showAbout, setShowAbout] = React.useState(false);

  const handleLogout = () => {
    logout();
    toast.success('Signed out');
    navigate('/login');
  };

  const avatarUrl = user
    ? `https://ui-avatars.com/api/?name=${encodeURIComponent(user.username)}&background=random`
    : '';

  return (
    <>
    <AnimatePresence>
      {showAbout && <AboutModal onClose={() => setShowAbout(false)} />}
    </AnimatePresence>
    <div className="min-h-screen pb-24 bg-slate-950 text-slate-200">
      {/* Header */}
      <header className="relative pt-8 pb-4 px-6 overflow-hidden">
        <div className="absolute top-0 right-0 w-1/2 h-full mood-gradient opacity-10 pointer-events-none" />
        
        <div className="max-w-7xl mx-auto relative z-10">
          <div className="flex items-center justify-between">
            <Link to="/" className="flex items-center gap-2 group">
              <div className="w-10 h-10 rounded-xl bg-brand-500 flex items-center justify-center shadow-lg shadow-brand-500/20 group-hover:scale-110 transition-transform">
                <Sparkles className="w-6 h-6 text-white" />
              </div>
              <h1 className="font-display font-bold text-2xl tracking-tight">Aura</h1>
            </Link>
            
            <div className="flex items-center gap-2 sm:gap-4">
              {user ? (
                <div className="flex items-center gap-4">
                  <div className="hidden sm:flex flex-col items-end">
                    <span className="text-sm font-bold text-white">{user.username}</span>
                    <span className="text-[10px] text-slate-500 uppercase tracking-widest">Member</span>
                  </div>
                  <div className="relative group">
                    <img
                      src={avatarUrl}
                      alt={user.username}
                      className="w-10 h-10 rounded-xl border border-white/10"
                    />
                    <button
                      onClick={handleLogout}
                      className="absolute -bottom-1 -right-1 p-1.5 rounded-lg bg-red-500 text-white shadow-lg opacity-0 group-hover:opacity-100 transition-opacity"
                    >
                      <LogOut className="w-3 h-3" />
                    </button>
                  </div>
                </div>
              ) : (
                <>
                  <Link
                    to="/login"
                    className="flex items-center gap-2 px-4 py-2 rounded-xl hover:bg-white/5 text-slate-400 hover:text-white transition-all font-medium"
                  >
                    <LogIn className="w-4 h-4" />
                    <span className="hidden sm:inline">Sign In</span>
                  </Link>
                  <Link
                    to="/register"
                    className="bg-white/5 hover:bg-white/10 border border-white/10 px-4 py-2 rounded-xl text-slate-200 transition-all font-medium"
                  >
                    Join
                  </Link>
                </>
              )}
              <div className="w-px h-6 bg-white/10 mx-2 hidden sm:block" />
              <button
                onClick={() => setShowAbout(true)}
                className="p-2 rounded-full hover:bg-white/5 text-slate-400 hover:text-slate-200 transition-colors hidden sm:block"
                aria-label="About Aura"
              >
                <Info className="w-5 h-5" />
              </button>
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-6 pt-8">
        <Outlet />
      </main>

      {/* Floating Bottom Nav */}
      {!isAuthPage && (
        <nav className="fixed bottom-6 left-1/2 -translate-x-1/2 glass-panel px-6 py-3 rounded-2xl flex items-center gap-8 z-50 shadow-2xl border-white/10">
          <Link
            to="/"
            className={`text-sm font-bold transition-colors ${location.pathname === '/' ? 'text-brand-500' : 'text-slate-400 hover:text-slate-200'}`}
          >
            Discover
          </Link>
          <Link
            to="/library"
            className={`text-sm font-bold transition-colors ${location.pathname === '/library' ? 'text-brand-500' : 'text-slate-400 hover:text-slate-200'}`}
          >
            Library
          </Link>
          <Link
            to="/assistant"
            className={`text-sm font-bold transition-colors ${location.pathname === '/assistant' ? 'text-brand-500' : 'text-slate-400 hover:text-slate-200'}`}
          >
            AI Assistant
          </Link>
          {user ? (
            <Link
              to="/stats"
              className={`text-sm font-bold transition-colors flex items-center gap-1.5 ${location.pathname === '/stats' ? 'text-brand-500' : 'text-slate-400 hover:text-slate-200'}`}
            >
              <BarChart2 className="w-4 h-4" />
              Stats
            </Link>
          ) : (
            <Link
              to="/login"
              className="text-slate-400 font-bold text-sm hover:text-slate-200 transition-colors flex items-center gap-1.5"
            >
              <LogIn className="w-4 h-4" />
              Sign In
            </Link>
          )}
        </nav>
      )}
    </div>
    </>
  );
};
