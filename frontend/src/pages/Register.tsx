import React, { useState, useEffect } from 'react';
import { motion } from 'motion/react';
import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Sparkles, Mail, Lock, User, ArrowRight } from 'lucide-react';
import { toast } from 'sonner';
import { useAuth } from '../contexts/AuthContext';

interface LocationState {
  from?: { pathname: string };
}

export default function Register() {
  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();
  const { user, register } = useAuth();

  const redirectTo = (location.state as LocationState | null)?.from?.pathname || '/';

  useEffect(() => {
    if (user) navigate(redirectTo, { replace: true });
  }, [user, navigate, redirectTo]);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (password.length < 6) {
      const message = 'Password must be at least 6 characters.';
      setError(message);
      toast.error(message);
      return;
    }
    setSubmitting(true);
    try {
      await register(username.trim(), email, password);
      toast.success(`Welcome to Aura, ${username.trim()}!`);
      navigate(redirectTo, { replace: true });
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Registration failed.';
      setError(message);
      toast.error(message);
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <div className="min-h-[80vh] flex items-center justify-center px-6">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        className="w-full max-w-md glass-panel rounded-3xl p-8 md:p-10 shadow-2xl"
      >
        <div className="flex flex-col items-center mb-8">
          <div className="w-12 h-12 rounded-2xl bg-brand-500 flex items-center justify-center shadow-lg shadow-brand-500/20 mb-4">
            <Sparkles className="w-7 h-7 text-white" />
          </div>
          <h1 className="font-display font-bold text-3xl tracking-tight">Join Aura</h1>
          <p className="text-slate-400 mt-2">Start your personalized leisure journey</p>
        </div>

        {error && (
          <div className="mb-6 p-4 rounded-xl bg-red-500/10 border border-red-500/20 text-red-400 text-sm text-center">
            {error}
          </div>
        )}

        <form onSubmit={handleSubmit} className="space-y-5">
          <div className="space-y-2">
            <label htmlFor="register-username" className="text-sm font-medium text-slate-300 ml-1">Username</label>
            <div className="relative group">
              <User className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-brand-500 transition-colors" />
              <input
                id="register-username"
                type="text"
                autoComplete="username"
                required
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-2xl py-3.5 pl-12 pr-4 text-slate-200 focus:outline-none focus:ring-2 focus:ring-brand-500/50 focus:bg-white/10 transition-all"
                placeholder="johndoe"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label htmlFor="register-email" className="text-sm font-medium text-slate-300 ml-1">Email Address</label>
            <div className="relative group">
              <Mail className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-brand-500 transition-colors" />
              <input
                id="register-email"
                type="email"
                autoComplete="email"
                required
                value={email}
                onChange={(e) => setEmail(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-2xl py-3.5 pl-12 pr-4 text-slate-200 focus:outline-none focus:ring-2 focus:ring-brand-500/50 focus:bg-white/10 transition-all"
                placeholder="name@example.com"
              />
            </div>
          </div>

          <div className="space-y-2">
            <label htmlFor="register-password" className="text-sm font-medium text-slate-300 ml-1">Password</label>
            <div className="relative group">
              <Lock className="absolute left-4 top-1/2 -translate-y-1/2 w-5 h-5 text-slate-500 group-focus-within:text-brand-500 transition-colors" />
              <input
                id="register-password"
                type="password"
                autoComplete="new-password"
                minLength={6}
                required
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                className="w-full bg-white/5 border border-white/10 rounded-2xl py-3.5 pl-12 pr-4 text-slate-200 focus:outline-none focus:ring-2 focus:ring-brand-500/50 focus:bg-white/10 transition-all"
                placeholder="At least 6 characters"
              />
            </div>
          </div>

          <button
            type="submit"
            disabled={submitting}
            className="w-full bg-brand-500 hover:bg-brand-600 text-white font-bold py-4 rounded-2xl shadow-lg shadow-brand-500/20 transition-all flex items-center justify-center gap-2 group disabled:opacity-60 disabled:cursor-not-allowed"
          >
            {submitting ? (
              <div className="w-5 h-5 border-2 border-white/30 border-t-white rounded-full animate-spin" />
            ) : (
              <>
                Create Account
                <ArrowRight className="w-5 h-5 group-hover:translate-x-1 transition-transform" />
              </>
            )}
          </button>
        </form>

        <div className="mt-8 text-center">
          <p className="text-slate-400 text-sm">
            Already have an account?{' '}
            <Link to="/login" className="text-brand-500 font-bold hover:text-brand-400 transition-colors">
              Sign In
            </Link>
          </p>
        </div>
      </motion.div>
    </div>
  );
}
