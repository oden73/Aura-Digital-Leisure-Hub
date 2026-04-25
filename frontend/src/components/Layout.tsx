import React from 'react';
import { Link, useLocation, Outlet, useNavigate } from 'react-router-dom';
import { Sparkles, Info, User, LogIn, LogOut } from 'lucide-react';
import { useAuth } from '../contexts/AuthContext';
import { logout } from '../firebase';

export const Layout: React.FC = () => {
  const location = useLocation();
  const navigate = useNavigate();
  const { user } = useAuth();
  const isAuthPage = ['/login', '/register'].includes(location.pathname);

  const handleLogout = async () => {
    await logout();
    navigate('/login');
  };

  return (
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
                    <span className="text-sm font-bold text-white">{user.displayName}</span>
                    <span className="text-[10px] text-slate-500 uppercase tracking-widest">Member</span>
                  </div>
                  <div className="relative group">
                    <img 
                      src={user.photoURL || `https://ui-avatars.com/api/?name=${user.displayName}`} 
                      alt={user.displayName || 'User'} 
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
              <button className="p-2 rounded-full hover:bg-white/5 text-slate-400 transition-colors hidden sm:block">
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
            <button className="text-slate-400 font-bold text-sm hover:text-slate-200 transition-colors flex items-center gap-2">
              <User className="w-4 h-4" />
              Profile
            </button>
          ) : (
            <Link 
              to="/login" 
              className="text-slate-400 font-bold text-sm hover:text-slate-200 transition-colors flex items-center gap-2"
            >
              <User className="w-4 h-4" />
              Sign In
            </Link>
          )}
        </nav>
      )}
    </div>
  );
};
