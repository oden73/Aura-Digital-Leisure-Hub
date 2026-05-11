import { MediaItem } from '../types';

/**
 * In dev, use same-origin requests so Vite can proxy `/v1` → API (avoids CORS).
 * In production builds, set VITE_API_URL to your deployed core API.
 */
const API_BASE = import.meta.env.DEV
  ? ''
  : ((import.meta.env.VITE_API_URL as string | undefined) ?? 'http://localhost:8080');

function apiURL(path: string): string {
  const p = path.startsWith('/') ? path : `/${path}`;
  if (!API_BASE) return p;
  return `${API_BASE.replace(/\/$/, '')}${p}`;
}

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
    const res = await fetch(apiURL('/v1/auth/refresh'), {
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

  let res: Response;
  try {
    res = await fetch(apiURL(path), { ...init, headers });
  } catch {
    throw new Error(
      import.meta.env.DEV
        ? 'Network error — is the Go API running on port 8080? (Vite proxies /v1 there in dev.)'
        : 'Network error — could not reach the API.',
    );
  }

  if (res.status === 401) {
    const ok = await doRefresh();
    if (ok) {
      headers.set('Authorization', `Bearer ${getAccessToken()!}`);
      try {
        res = await fetch(apiURL(path), { ...init, headers });
      } catch {
        throw new Error(
          import.meta.env.DEV
            ? 'Network error — is the Go API running on port 8080?'
            : 'Network error — could not reach the API.',
        );
      }
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

function steamUrlFromCover(coverUrl?: string): string | undefined {
  const match = coverUrl?.match(/\/steam\/apps\/(\d+)\//);
  return match ? `https://store.steampowered.com/app/${match[1]}` : undefined;
}

export function mapApiItem(item: ApiItem & { match_score?: number; match_reason?: string }): MediaItem {
  return {
    id: item.id,
    type: item.media_type === 'cinema' ? 'movie' : item.media_type,
    title: item.title,
    image: item.cover_image_url || `https://picsum.photos/seed/${encodeURIComponent(item.id)}/400/600`,
    externalUrl: steamUrlFromCover(item.cover_image_url),
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

// ---- Stats ----

export interface ApiMediaTypeStats {
  media_type: string;
  total: number;
  rated: number;
  avg_rating: number;
  favorites: number;
  completed: number;
}

export interface ApiUserStats {
  total_interactions: number;
  rated_count: number;
  avg_rating: number;
  favorite_count: number;
  completed_count: number;
  by_media_type: ApiMediaTypeStats[];
}

export async function apiGetStats(): Promise<ApiUserStats> {
  const res = await apiFetch('/v1/profile/stats');
  if (!res.ok) throw new Error('Failed to fetch stats');
  return res.json();
}

// ---- External Accounts ----

export async function apiLinkExternalAccount(
  serviceName: string,
  externalUserId: string,
  externalProfileUrl?: string,
): Promise<void> {
  const res = await apiFetch('/v1/external-accounts', {
    method: 'POST',
    body: JSON.stringify({
      service_name: serviceName,
      external_user_id: externalUserId,
      ...(externalProfileUrl ? { external_profile_url: externalProfileUrl } : {}),
    }),
  });
  if (!res.ok) {
    let msg = 'Failed to link account';
    try { const e = await res.json(); msg = e.message ?? msg; } catch { /* ignore */ }
    throw new Error(msg);
  }
}

// ---- Recommendations ----

export interface ApiRecommendationItem extends ApiItem {
  match_score?: number;
  match_reason?: string;
}

type ApiRecommendationSummary = Partial<ApiRecommendationItem> & {
  item_id?: string;
  score?: number;
};

function isFullApiItem(item: ApiRecommendationSummary): item is ApiRecommendationItem {
  return Boolean(item.id && item.media_type);
}

export async function apiGetRecommendations(
  limit = 20,
  moods?: string[],
): Promise<ApiRecommendationItem[]> {
  const filters: Record<string, unknown> = {};
  if (moods && moods.length > 0) filters.moods = moods;
  try {
    const raw = localStorage.getItem('aura_preferred_genres');
    if (raw) {
      const genres: string[] = JSON.parse(raw);
      if (Array.isArray(genres) && genres.length > 0) filters.genres = genres;
    }
  } catch { /* ignore */ }
  const res = await apiFetch('/v1/recommendations', {
    method: 'POST',
    body: JSON.stringify({ filters }),
  });
  if (!res.ok) return [];
  const raw = await res.json();
  const summaries: ApiRecommendationSummary[] = Array.isArray(raw)
    ? raw
    : (raw.items ?? raw.recommendations ?? []);

  const enriched = await Promise.allSettled(
    summaries.slice(0, limit).map(async summary => {
      const id = summary.id ?? summary.item_id;
      if (!id) return null;

      const match_score = summary.match_score ?? summary.score;
      const match_reason = summary.match_reason;

      if (isFullApiItem(summary) && summary.cover_image_url && summary.criteria?.tonality) {
        return { ...summary, match_score, match_reason };
      }

      try {
        const full = await apiGetContent(id);
        return { ...full, match_score, match_reason };
      } catch {
        if (isFullApiItem(summary)) {
          return { ...summary, match_score, match_reason };
        }
        return null;
      }
    }),
  );

  return enriched
    .flatMap(result => result.status === 'fulfilled' && result.value ? [result.value] : []);
}
