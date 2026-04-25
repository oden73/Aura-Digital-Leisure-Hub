import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'motion/react';
import { ArrowLeft, Star, Gamepad2, Book, Film, Share2, Heart, ExternalLink, Plus, Check } from 'lucide-react';
import { toast } from 'sonner';
import { MOCK_DATA } from '../data';
import { useAuth } from '../contexts/AuthContext';
import { db, handleFirestoreError, OperationType } from '../firebase';
import { doc, setDoc, deleteDoc, onSnapshot, serverTimestamp } from 'firebase/firestore';

export default function ContentDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();
  const [isInLibrary, setIsInLibrary] = useState(false);
  const [isAdding, setIsAdding] = useState(false);
  
  const item = MOCK_DATA.find(i => i.id === id);

  useEffect(() => {
    if (!user || !id) return;

    const libRef = doc(db, 'users', user.uid, 'library', id);
    const unsubscribe = onSnapshot(libRef, (doc) => {
      setIsInLibrary(doc.exists());
    }, (error) => {
      handleFirestoreError(error, OperationType.GET, `users/${user.uid}/library/${id}`);
    });

    return unsubscribe;
  }, [user, id]);

  if (!item) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center">
        <h2 className="text-2xl font-bold mb-4">Content not found</h2>
        <button 
          onClick={() => navigate('/')}
          className="text-brand-500 hover:underline"
        >
          Back to home
        </button>
      </div>
    );
  }

  const toggleLibrary = async () => {
    if (!user) {
      navigate('/login');
      return;
    }

    setIsAdding(true);
    const libRef = doc(db, 'users', user.uid, 'library', item.id);
    const wasInLibrary = isInLibrary;

    try {
      if (wasInLibrary) {
        await deleteDoc(libRef);
        toast.success(`Removed “${item.title}” from your library`);
      } else {
        await setDoc(libRef, {
          itemId: item.id,
          type: item.type,
          title: item.title,
          addedAt: serverTimestamp(),
          status: 'to-watch'
        });
        toast.success(`Added “${item.title}” to your library`);
      }
    } catch (error) {
      handleFirestoreError(
        error,
        OperationType.WRITE,
        `users/${user.uid}/library/${item.id}`,
        wasInLibrary
          ? `Couldn't remove “${item.title}” from your library.`
          : `Couldn't add “${item.title}” to your library.`,
      );
    } finally {
      setIsAdding(false);
    }
  };

  const TypeIcon = () => {
    switch (item.type) {
      case 'game': return <Gamepad2 className="w-6 h-6" />;
      case 'book': return <Book className="w-6 h-6" />;
      case 'movie': return <Film className="w-6 h-6" />;
    }
  };

  return (
    <div className="max-w-6xl mx-auto px-6 py-8">
      <button 
        onClick={() => navigate(-1)}
        className="flex items-center gap-2 text-slate-400 hover:text-white transition-colors mb-8 group"
      >
        <ArrowLeft className="w-5 h-5 group-hover:-translate-x-1 transition-transform" />
        Back
      </button>

      <div className="grid md:grid-cols-[400px_1fr] gap-12">
        {/* Left: Image & Quick Stats */}
        <motion.div 
          initial={{ opacity: 0, x: -20 }}
          animate={{ opacity: 1, x: 0 }}
          className="space-y-6"
        >
          <div className="aspect-[2/3] rounded-3xl overflow-hidden glass-panel shadow-2xl">
            <img 
              src={item.image} 
              alt={item.title} 
              className="w-full h-full object-cover"
              referrerPolicy="no-referrer"
            />
          </div>
          
          <div className="grid grid-cols-2 gap-4">
            <div className="glass-panel p-4 rounded-2xl text-center">
              <p className="text-xs text-slate-500 uppercase font-bold tracking-wider mb-1">Rating</p>
              <div className="flex items-center justify-center gap-1 text-yellow-400 font-bold text-xl">
                <Star className="w-5 h-5 fill-yellow-400" />
                {item.rating}
              </div>
            </div>
            <div className="glass-panel p-4 rounded-2xl text-center">
              <p className="text-xs text-slate-500 uppercase font-bold tracking-wider mb-1">Type</p>
              <div className="flex items-center justify-center gap-2 text-white font-bold text-lg">
                <TypeIcon />
                <span className="capitalize">{item.type}</span>
              </div>
            </div>
          </div>

          <div className="flex flex-col gap-3">
            <button 
              onClick={toggleLibrary}
              disabled={isAdding}
              className={`w-full font-bold py-4 rounded-2xl shadow-lg transition-all flex items-center justify-center gap-2 ${
                isInLibrary 
                  ? 'bg-green-500/10 text-green-500 border border-green-500/20 hover:bg-green-500/20' 
                  : 'bg-brand-500 hover:bg-brand-600 text-white shadow-brand-500/20'
              }`}
            >
              {isAdding ? (
                <div className="w-5 h-5 border-2 border-white/20 border-t-white rounded-full animate-spin" />
              ) : isInLibrary ? (
                <>
                  <Check className="w-5 h-5" />
                  In Library
                </>
              ) : (
                <>
                  <Plus className="w-5 h-5" />
                  Add to Library
                </>
              )}
            </button>
            <button className="w-full bg-white/5 hover:bg-white/10 text-white font-bold py-4 rounded-2xl border border-white/10 transition-all flex items-center justify-center gap-2">
              <ExternalLink className="w-5 h-5" />
              Open in {item.type === 'game' ? 'Steam' : item.type === 'book' ? 'LiveLib' : 'Кинопоиск'}
            </button>
          </div>
        </motion.div>

        {/* Right: Details */}
        <motion.div 
          initial={{ opacity: 0, x: 20 }}
          animate={{ opacity: 1, x: 0 }}
          className="space-y-8"
        >
          <div className="flex justify-between items-start">
            <div>
              <h1 className="font-display font-bold text-5xl mb-4 tracking-tight">{item.title}</h1>
              <div className="flex flex-wrap gap-2">
                <span className="px-3 py-1 rounded-full bg-brand-500/10 text-brand-400 text-sm font-semibold border border-brand-500/20">
                  {item.tonality}
                </span>
                <span className="px-3 py-1 rounded-full bg-white/5 text-slate-300 text-sm font-medium border border-white/10">
                  {item.setting}
                </span>
                {item.genre.map(g => (
                  <span key={g} className="px-3 py-1 rounded-full bg-white/5 text-slate-400 text-sm border border-white/10">
                    {g}
                  </span>
                ))}
              </div>
            </div>
            <div className="flex gap-2">
              <button className="p-3 rounded-xl glass-panel hover:bg-white/10 transition-colors text-slate-400 hover:text-red-500">
                <Heart className="w-6 h-6" />
              </button>
              <button className="p-3 rounded-xl glass-panel hover:bg-white/10 transition-colors text-slate-400 hover:text-white">
                <Share2 className="w-6 h-6" />
              </button>
            </div>
          </div>

          <div className="space-y-4">
            <h3 className="text-xl font-bold font-display">About this {item.type}</h3>
            <p className="text-slate-400 leading-relaxed text-lg">
              Experience a unique journey through {item.setting.toLowerCase()} landscapes. 
              This {item.type} explores themes of {item.themes.join(', ').toLowerCase()}, 
              delivering a {item.tonality.toLowerCase()} atmosphere that resonates with {item.targetAudience.toLowerCase()}.
            </p>
          </div>

          <div className="grid sm:grid-cols-2 gap-8 pt-8 border-t border-white/5">
            <div className="space-y-2">
              <h4 className="text-sm font-bold text-slate-500 uppercase tracking-widest">Themes</h4>
              <div className="flex flex-wrap gap-2">
                {item.themes.map(theme => (
                  <span key={theme} className="text-slate-200 font-medium">{theme}</span>
                ))}
              </div>
            </div>
            <div className="space-y-2">
              <h4 className="text-sm font-bold text-slate-500 uppercase tracking-widest">Details</h4>
              <div className="space-y-1">
                <p className="text-slate-200"><span className="text-slate-500">Volume:</span> {item.volume}</p>
                {item.platform && (
                  <p className="text-slate-200"><span className="text-slate-500">Platforms:</span> {item.platform.join(', ')}</p>
                )}
                <p className="text-slate-200"><span className="text-slate-500">Audience:</span> {item.targetAudience}</p>
              </div>
            </div>
          </div>

          {/* Cross-Media Connections */}
          <div className="pt-12">
            <h3 className="text-2xl font-bold font-display mb-6">Cross-Media Connections</h3>
            <div className="glass-panel p-6 rounded-3xl border-brand-500/20 bg-brand-500/5">
              <p className="text-slate-300 italic mb-4">
                "Because you enjoyed the {item.tonality.toLowerCase()} atmosphere and {item.setting.toLowerCase()} setting of {item.title}, 
                Aura suggests exploring these related experiences..."
              </p>
              <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                {MOCK_DATA.filter(i => i.id !== item.id && (i.tonality === item.tonality || i.setting === item.setting)).slice(0, 3).map(related => (
                  <div 
                    key={related.id}
                    onClick={() => navigate(`/content/${related.id}`)}
                    className="cursor-pointer group"
                  >
                    <div className="aspect-[2/3] rounded-xl overflow-hidden mb-2">
                      <img src={related.image} alt={related.title} className="w-full h-full object-cover group-hover:scale-110 transition-transform" referrerPolicy="no-referrer" />
                    </div>
                    <p className="text-xs font-bold truncate">{related.title}</p>
                  </div>
                ))}
              </div>
            </div>
          </div>
        </motion.div>
      </div>
    </div>
  );
}
