import React, { useState, useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { motion } from 'motion/react';
import { ArrowLeft, Star, Gamepad2, Book, Film, Share2, Heart, ExternalLink, Plus, Check } from 'lucide-react';
import { toast } from 'sonner';
import { useAuth } from '../contexts/AuthContext';
import {
  apiGetContent,
  apiGetLibrary,
  apiUpdateInteraction,
  apiSearch,
  mapApiItem,
  ApiItem,
  ApiInteraction,
} from '../services/api';
import { MediaItem } from '../types';

export default function ContentDetail() {
  const { id } = useParams();
  const navigate = useNavigate();
  const { user } = useAuth();

  const [item, setItem] = useState<MediaItem | null>(null);
  const [loading, setLoading] = useState(true);
  const [interaction, setInteraction] = useState<ApiInteraction | null>(null);
  const [isInLibrary, setIsInLibrary] = useState(false);
  const [isAdding, setIsAdding] = useState(false);
  const [userRating, setUserRating] = useState<number>(0);
  const [hoverRating, setHoverRating] = useState<number>(0);
  const [isRating, setIsRating] = useState(false);
  const [related, setRelated] = useState<MediaItem[]>([]);

  useEffect(() => {
    if (!id) return;
    setLoading(true);

    apiGetContent(id)
      .then((raw: ApiItem) => {
        const mapped = mapApiItem(raw);
        setItem(mapped);

        const query = raw.criteria.tonality || raw.criteria.setting || raw.title;
        apiSearch(query, 6)
          .then(results => {
            setRelated(results.map(mapApiItem).filter(r => r.id !== id).slice(0, 3));
          })
          .catch(() => {});
      })
      .catch(() => setItem(null))
      .finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!user || !id) return;
    apiGetLibrary()
      .then(interactions => {
        const found = interactions.find(i => i.item_id === id);
        if (found) {
          setInteraction(found);
          setIsInLibrary(found.status && found.status !== 'dropped');
          setUserRating(found.rating ?? 0);
        }
      })
      .catch(() => {});
  }, [user, id]);

  if (loading) {
    return (
      <div className="min-h-[60vh] flex items-center justify-center">
        <div className="w-12 h-12 border-4 border-brand-500/20 border-t-brand-500 rounded-full animate-spin" />
      </div>
    );
  }

  if (!item) {
    return (
      <div className="min-h-[60vh] flex flex-col items-center justify-center">
        <h2 className="text-2xl font-bold mb-4">Content not found</h2>
        <button onClick={() => navigate('/')} className="text-brand-500 hover:underline">
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
    if (!id) return;

    setIsAdding(true);
    const wasInLibrary = isInLibrary;
    try {
      await apiUpdateInteraction(id, { status: wasInLibrary ? 'dropped' : 'planned' });
      setIsInLibrary(!wasInLibrary);
      if (wasInLibrary) {
        setInteraction(prev => prev ? { ...prev, status: 'dropped' } : null);
      } else {
        setInteraction(prev => prev
          ? { ...prev, status: 'planned' }
          : { id: 0, user_id: '', item_id: id, status: 'planned', is_favorite: false, updated_at: '' },
        );
      }
      toast.success(
        wasInLibrary
          ? `Removed "${item.title}" from your library`
          : `Added "${item.title}" to your library`,
      );
    } catch {
      toast.error(wasInLibrary ? `Couldn't remove "${item.title}" from your library.` : `Couldn't add "${item.title}" to your library.`);
    } finally {
      setIsAdding(false);
    }
  };

  const submitRating = async (stars: number) => {
    if (!user) { navigate('/login'); return; }
    if (!id) return;
    setIsRating(true);
    try {
      if (!isInLibrary) {
        await apiUpdateInteraction(id, { status: 'completed', rating: stars });
        setIsInLibrary(true);
        setInteraction(prev => prev
          ? { ...prev, status: 'completed', rating: stars }
          : { id: 0, user_id: '', item_id: id, status: 'completed', rating: stars, is_favorite: false, updated_at: '' },
        );
      } else {
        await apiUpdateInteraction(id, { rating: stars });
        setInteraction(prev => prev ? { ...prev, rating: stars } : null);
      }
      setUserRating(stars);
      toast.success(`Rated "${item.title}" ${stars}/5`);
    } catch {
      toast.error('Failed to save rating');
    } finally {
      setIsRating(false);
    }
  };

  const TypeIcon = () => {
    switch (item.type) {
      case 'game':  return <Gamepad2 className="w-6 h-6" />;
      case 'book':  return <Book className="w-6 h-6" />;
      case 'movie': return <Film className="w-6 h-6" />;
    }
  };

  const displayRating = hoverRating || userRating;

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

          {/* User Rating */}
          <div className="glass-panel p-4 rounded-2xl">
            <p className="text-xs text-slate-500 uppercase font-bold tracking-wider mb-3 text-center">Your Rating</p>
            <div
              className={`flex items-center justify-center gap-1 ${isRating ? 'opacity-50 pointer-events-none' : ''}`}
              onMouseLeave={() => setHoverRating(0)}
            >
              {[1, 2, 3, 4, 5].map(star => (
                <button
                  key={star}
                  onClick={() => submitRating(star)}
                  onMouseEnter={() => setHoverRating(star)}
                  className="p-1 transition-transform hover:scale-110"
                  aria-label={`Rate ${star} stars`}
                >
                  <Star
                    className={`w-7 h-7 transition-colors ${
                      star <= displayRating
                        ? 'text-yellow-400 fill-yellow-400'
                        : 'text-slate-600'
                    }`}
                  />
                </button>
              ))}
            </div>
            {userRating > 0 && (
              <p className="text-center text-xs text-slate-500 mt-2">{userRating}/5</p>
            )}
            {!user && (
              <p className="text-center text-xs text-slate-500 mt-2">
                <button onClick={() => navigate('/login')} className="text-brand-500 hover:underline">
                  Sign in
                </button>{' '}
                to rate
              </p>
            )}
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
              {interaction && interaction.status && interaction.status !== 'dropped' && (
                <div className="mt-3">
                  <span className="px-3 py-1 rounded-full bg-green-500/10 text-green-400 text-xs font-bold border border-green-500/20 uppercase tracking-wider">
                    {interaction.status}
                  </span>
                </div>
              )}
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
                {item.volume && (
                  <p className="text-slate-200"><span className="text-slate-500">Volume:</span> {item.volume}</p>
                )}
                {item.platform && (
                  <p className="text-slate-200"><span className="text-slate-500">Platforms:</span> {item.platform.join(', ')}</p>
                )}
                <p className="text-slate-200"><span className="text-slate-500">Audience:</span> {item.targetAudience}</p>
              </div>
            </div>
          </div>

          {/* Cross-Media Connections */}
          {related.length > 0 && (
            <div className="pt-12">
              <h3 className="text-2xl font-bold font-display mb-6">Cross-Media Connections</h3>
              <div className="glass-panel p-6 rounded-3xl border-brand-500/20 bg-brand-500/5">
                <p className="text-slate-300 italic mb-4">
                  "Because you enjoyed the {item.tonality.toLowerCase()} atmosphere and {item.setting.toLowerCase()} setting of {item.title},
                  Aura suggests exploring these related experiences..."
                </p>
                <div className="grid grid-cols-2 sm:grid-cols-3 gap-4">
                  {related.map(r => (
                    <div
                      key={r.id}
                      onClick={() => navigate(`/content/${r.id}`)}
                      className="cursor-pointer group"
                    >
                      <div className="aspect-[2/3] rounded-xl overflow-hidden mb-2">
                        <img src={r.image} alt={r.title} className="w-full h-full object-cover group-hover:scale-110 transition-transform" referrerPolicy="no-referrer" />
                      </div>
                      <p className="text-xs font-bold truncate">{r.title}</p>
                    </div>
                  ))}
                </div>
              </div>
            </div>
          )}
        </motion.div>
      </div>
    </div>
  );
}
