import React from 'react';
import { motion } from 'motion/react';
import { Link } from 'react-router-dom';
import { Sparkles, Home, AlertCircle } from 'lucide-react';

export default function NotFound() {
  return (
    <div className="min-h-[80vh] flex flex-col items-center justify-center px-6 text-center">
      <motion.div
        initial={{ opacity: 0, scale: 0.9 }}
        animate={{ opacity: 1, scale: 1 }}
        className="space-y-8"
      >
        <div className="relative inline-block">
          <div className="w-24 h-24 rounded-3xl bg-red-500/10 flex items-center justify-center border border-red-500/20 mx-auto">
            <AlertCircle className="w-12 h-12 text-red-500" />
          </div>
          <motion.div
            animate={{ 
              scale: [1, 1.2, 1],
              rotate: [0, 10, -10, 0]
            }}
            transition={{ repeat: Infinity, duration: 4 }}
            className="absolute -top-2 -right-2 w-8 h-8 rounded-full bg-brand-500 flex items-center justify-center shadow-lg"
          >
            <Sparkles className="w-4 h-4 text-white" />
          </motion.div>
        </div>

        <div className="space-y-4">
          <h1 className="font-display font-bold text-6xl md:text-8xl tracking-tighter text-white">404</h1>
          <h2 className="text-2xl md:text-3xl font-bold text-slate-300">Lost in the Digital Void?</h2>
          <p className="text-slate-500 max-w-md mx-auto text-lg">
            The page you're looking for has drifted beyond our reach. Let's get you back to the hub.
          </p>
        </div>

        <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
          <Link 
            to="/"
            className="flex items-center gap-2 bg-brand-500 hover:bg-brand-600 text-white font-bold px-8 py-4 rounded-2xl transition-all shadow-lg shadow-brand-500/20"
          >
            <Home className="w-5 h-5" />
            Back to Hub
          </Link>
          <button 
            onClick={() => window.location.reload()}
            className="px-8 py-4 rounded-2xl glass-panel hover:bg-white/10 text-slate-300 font-bold transition-all"
          >
            Retry Connection
          </button>
        </div>
      </motion.div>
    </div>
  );
}
