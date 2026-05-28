/**
 * Auth utilities for making authenticated API requests.
 * Provides helper functions that attach the session JWT token
 * to outgoing requests.
 */

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

/**
 * Get the stored session token from localStorage.
 * Returns null during SSR or when not authenticated.
 */
export function getToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('nomados-session');
}

/**
 * Build headers with the Authorization Bearer token.
 */
export function authHeaders(): HeadersInit {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };
  const token = getToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

/**
 * Make an authenticated GET request.
 */
export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const url = path.startsWith('http') ? path : `${GATEWAY_URL}${path}`;
  return fetch(url, {
    ...init,
    headers: {
      ...authHeaders(),
      ...init?.headers,
    },
  });
}

/**
 * Make an authenticated POST request with a JSON body.
 */
export async function apiPost(path: string, body?: unknown): Promise<Response> {
  const url = path.startsWith('http') ? path : `${GATEWAY_URL}${path}`;
  return fetch(url, {
    method: 'POST',
    headers: authHeaders(),
    body: body ? JSON.stringify(body) : undefined,
  });
}