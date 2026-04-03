import { get } from 'svelte/store';
import { authToken } from './stores';

const API_BASE = '/api';

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const token = get(authToken);
  const headers: Record<string, string> = {
    ...(options.headers as Record<string, string>)
  };

  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  if (options.body && typeof options.body === 'string') {
    headers['Content-Type'] = 'application/json';
  }

  headers['X-Timezone'] = Intl.DateTimeFormat().resolvedOptions().timeZone;

  const res = await fetch(`${API_BASE}${path}`, {
    ...options,
    headers,
    credentials: 'include'
  });

  if (res.status === 401) {
    // Attempt token refresh before giving up
    try {
      const refreshed = await fetch(`${API_BASE}/auth/refresh`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include'
      });
      if (refreshed.ok) {
        const refreshData = await refreshed.json();
        if (refreshData.data?.access_token) {
          authToken.set(refreshData.data.access_token);
          // Retry the original request with new token
          headers['Authorization'] = `Bearer ${refreshData.data.access_token}`;
          const retryRes = await fetch(`${API_BASE}${path}`, {
            ...options,
            headers,
            credentials: 'include'
          });
          if (retryRes.ok) {
            if (retryRes.status === 204 || retryRes.headers.get('content-length') === '0') {
              return undefined as T;
            }
            const retryContentType = retryRes.headers.get('content-type') || '';
            if (!retryContentType.includes('application/json')) {
              return await retryRes.text() as unknown as T;
            }
            const retryData = await retryRes.json();
            return retryData.data;
          }
        }
      }
    } catch {}
    // Refresh failed — redirect to login
    authToken.set(null);
    window.location.href = '/login';
    throw new Error('Unauthorized');
  }

  if (!res.ok) {
    let message = `Request failed with status ${res.status}`;
    try {
      const data = await res.json();
      message = data.error || message;
    } catch {}
    throw new Error(message);
  }

  if (res.status === 204 || res.headers.get('content-length') === '0') {
    return undefined as T;
  }

  const contentType = res.headers.get('content-type') || '';
  if (!contentType.includes('application/json')) {
    return await res.text() as unknown as T;
  }

  const data = await res.json();
  return data.data;
}

export const api = {
  get: <T>(path: string) => request<T>(path),

  post: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: 'POST',
      body: body ? JSON.stringify(body) : undefined
    }),

  put: <T>(path: string, body?: unknown) =>
    request<T>(path, {
      method: 'PUT',
      body: body ? JSON.stringify(body) : undefined
    }),

  del: <T>(path: string) => request<T>(path, { method: 'DELETE' })
};
