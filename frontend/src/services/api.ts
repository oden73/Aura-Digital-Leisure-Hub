import { MediaItem } from '../types';

const BASE = (import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080';

const KEY_ACCESS  = 'aura_access';
const KEY_REFRESH = 'aura_refresh';

export function getAccessToken(): string | null {
  return localStorage.getItem(KEY_ACCESS);
}

export function saveTokens(access: string, refresh: string): void {
  localStorage.setItem(KEY_ACCESS, access);
  localStorage.setItem(KEY_REFRESH, refresh);
}

export function clearTokens(): void {
  localStorage.removeItem(KEY_ACCESS);
  localStorage.removeItem(KEY_REFRESH);
}

async function doRefresh(): Promise<boolean> {
  const refresh = localStorage.getItem(KEY_REFRESH);
  if (!refresh) return false;
  try {
    const res = await fetch(`${BASE}/v1/auth/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh }),
    });
    if (!res.ok) { clearTokens(); return false; }
    const data: { access: string; refresh: string } = await res.json();
    saveTokens(data.access, data.refresh);
    return true;
  } catch {
    clearTokens();
    return false;
  }
}

export async function apiFetch(path: string, init: RequestInit = {}): Promise<Response> {
  const headers = new Headers(init.headers);
  const token = getAccessToken();
  if (token) headers.set('Authorization', `Bearer ${token}`);
  if (init.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json');

  let res = await fetch(`${BASE}${path}`, { ...init, headers });

  if (res.status === 401) {
    const ok = await doRefresh();
    if (ok) {
      headers.set('Authorization', `Bearer ${getAccessToken()!}`);
      res = await fetch(`${BASE}${path}`, { ...init, headers });
    }
  }
  return res;
}

// ---- Auth ----

export async function apiLogin(email: string, password: string): Promise<void> {
  const res = await apiFetch('/v1/auth/login', {
    method: 'POST',
    body: JSON.stringify({ email, password }),
  });
  if (!res.ok) throw new Error('Invalid credentials');
  const data: { access: string; refresh: string } = await res.json();
  saveTokens(data.access, data.refresh);
}

export async function apiRegister(username: string, email: string, password: string): Promise<void> {
  const res = await apiFetch('/v1/auth/register', {
    method: 'POST',
    body: JSON.stringify({ username, email, password }),
  });
  if (!res.ok) {
    let msg = 'Registration failed';
    try { const e = await res.json(); msg = e.message ?? msg; } catch { /* ignore */ }
    throw new Error(msg);
  }
  const data: { access: string; refresh: string } = await res.json();
  saveTokens(data.access, data.refresh);
}

export interface ApiProfile {
  id: string;
  username: string;
  email: string;
  created_at: string;
}

export async function apiProfile(): Promise<ApiProfile> {
  const res = await apiFetch('/v1/profile');
  if (!res.ok) throw new Error('Unauthorized');
  return res.json();
}

// ---- Content ----

export interface ApiItem {
  id: string;
  title: string;
  original_title?: string;
  description?: string;
  release_date: string | null;
  cover_image_url?: string;
  average_rating: number;
  media_type: 'game' | 'cinema' | 'book';
  criteria: {
    genre?: string;
    setting?: string;
    themes?: string;
    tonality?: string;
    target_audience?: string;
  };
}

export function mapApiItem(item: ApiItem & { match_score?: number; match_reason?: string }): MediaItem {
  return {
    id: item.id,
    type: item.media_type === 'cinema' ? 'movie' : item.media_type,
    title: item.title,
    image: item.cover_image_url || `https://picsum.photos/seed/${encodeURIComponent(item.id)}/400/600`,
    genre: item.criteria?.genre
      ? item.criteria.genre.split(',').map(s => s.trim()).filter(Boolean)
      : [],
    setting: item.criteria?.setting ?? '',
    themes: item.criteria?.themes
      ? item.criteria.themes.split(',').map(s => s.trim()).filter(Boolean)
      : [],
    tonality: item.criteria?.tonality ?? '',
    targetAudience: item.criteria?.target_audience ?? '',
    rating: item.average_rating,
    matchScore: item.match_score,
    matchReason: item.match_reason,
  };
}

export async function apiSearch(q: string, limit = 20): Promise<ApiItem[]> {
  const params = new URLSearchParams();
  if (q) params.set('q', q);
  params.set('limit', String(limit));
  const res = await apiFetch(`/v1/search?${params}`);
  if (!res.ok) throw new Error('Search failed');
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function apiGetContent(id: string): Promise<ApiItem> {
  const res = await apiFetch(`/v1/content/${encodeURIComponent(id)}`);
  if (!res.ok) throw new Error('Not found');
  return res.json();
}

// ---- Library & Interactions ----

export interface ApiInteraction {
  id: number;
  user_id: string;
  item_id: string;
  status: string;
  rating?: number;
  is_favorite: boolean;
  review_text?: string;
  updated_at: string;
}

export async function apiGetLibrary(): Promise<ApiInteraction[]> {
  const res = await apiFetch('/v1/library');
  if (!res.ok) return [];
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export interface ApiLibraryItem {
  interaction: ApiInteraction;
  item: ApiItem;
}

export async function apiGetLibraryItems(limit = 100): Promise<ApiLibraryItem[]> {
  const res = await apiFetch(`/v1/library/items?limit=${limit}`);
  if (!res.ok) return [];
  const data = await res.json();
  return Array.isArray(data) ? data : [];
}

export async function apiUpdateInteraction(
  itemId: string,
  data: { status?: string; rating?: number; is_favorite?: boolean; review_text?: string },
): Promise<void> {
  const res = await apiFetch('/v1/interactions', {
    method: 'PUT',
    body: JSON.stringify({ item_id: itemId, data }),
  });
  if (!res.ok) throw new Error('Failed to update interaction');
}

// ---- Recommendations ----

export interface ApiRecommendationItem extends ApiItem {
  match_score?: number;
  match_reason?: string;
}

export async function apiGetRecommendations(limit = 20): Promise<ApiRecommendationItem[]> {
  const res = await apiFetch('/v1/recommendations', {
    method: 'POST',
    body: JSON.stringify({ limit }),
  });
  if (!res.ok) return [];
  const raw = await res.json();
  const items: ApiRecommendationItem[] = Array.isArray(raw)
    ? raw
    : (raw.items ?? raw.recommendations ?? []);

  const needsEnrich = items.filter(i => !i.cover_image_url || !i.criteria?.tonality);
  if (needsEnrich.length > 0) {
    await Promise.allSettled(
      needsEnrich.map(async i => {
        try {
          const full = await apiGetContent(i.id);
          if (!i.cover_image_url) i.cover_image_url = full.cover_image_url;
          if (!i.criteria) i.criteria = {};
          if (!i.criteria.tonality) i.criteria.tonality = full.criteria?.tonality;
        } catch { /* ignore */ }
      }),
    );
  }

  return items;
}
