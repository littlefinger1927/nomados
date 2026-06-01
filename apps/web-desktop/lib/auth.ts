/**
 * Auth utilities for NomadOS web desktop client.
 *
 * Provides client-side key derivation for the vault service and
 * authenticated API request helpers.
 */

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || (typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'https://localhost' : 'http://localhost:8080');

/** Store session token in both HttpOnly cookie and localStorage (migration period). */
export function setSessionToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('nomados-session', token);
  // HttpOnly cookies cannot be set from JS — use SameSite=Strict + Secure as fallback.
  // The server should set the HttpOnly cookie via Set-Cookie header on login responses.
  document.cookie = `nomados-session=${token}; path=/; max-age=${60 * 60 * 24 * 7}; SameSite=Strict; Secure`;
}

/** Store refresh token in both cookie and localStorage. */
export function setRefreshToken(token: string): void {
  if (typeof window === 'undefined') return;
  localStorage.setItem('nomados-refresh', token);
  document.cookie = `nomados-refresh=${token}; path=/; max-age=${60 * 60 * 24 * 30}; SameSite=Strict; Secure`;
}

/** Get refresh token from localStorage (primary) or cookie (fallback). */
export function getRefreshToken(): string | null {
  if (typeof window === 'undefined') return null;
  const stored = localStorage.getItem('nomados-refresh');
  if (stored) return stored;
  const match = document.cookie.match(/nomados-refresh=([^;]+)/);
  return match ? match[1] : null;
}

/** Get session token, checking localStorage first (since HttpOnly cookies are invisible to JS), then cookie fallback. */
export function getSessionToken(): string | null {
  if (typeof window === 'undefined') return null;
  // localStorage is primary — the server sets HttpOnly cookies as a backup
  const stored = localStorage.getItem('nomados-session');
  if (stored) return stored;
  // Fallback to non-HttpOnly cookie for migration
  const match = document.cookie.match(/nomados-session=([^;]+)/);
  return match ? match[1] : null;
}

/** Logout: clear all tokens, attempt session revocation, redirect to login. */
export async function logout(): Promise<void> {
  const { clearCachedMasterKey } = await import('./keys');
  clearCachedMasterKey();

  const token = getSessionToken();
  if (token) {
    try {
      await fetch(`${GATEWAY_URL}/v1/session/revoke`, {
        method: 'POST',
        headers: { 'Authorization': `Bearer ${token}` },
      });
    } catch {
      // Best-effort — don't block logout on network failure
    }
  }

  // Clear localStorage
  localStorage.removeItem('nomados-session');
  localStorage.removeItem('nomados-refresh');

  // Clear cookies with Secure flag matching how they were set
  document.cookie = 'nomados-session=; path=/; max-age=0; SameSite=Strict; Secure';
  document.cookie = 'nomados-refresh=; path=/; max-age=0; SameSite=Strict; Secure';

  // Redirect to login
  if (typeof window !== 'undefined') {
    window.location.href = '/login';
  }
}

/**
 * Build headers with the Authorization Bearer token.
 */
export function authHeaders(): HeadersInit {
  const headers: HeadersInit = {
    'Content-Type': 'application/json',
  };
  const token = getSessionToken();
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  return headers;
}

/**
 * Derive a 256-bit master key from a passphrase and salt using PBKDF2.
 *
 * Phase 5 browser-compatible alternative to Argon2id.
 * Phase 6 will add WASM-based Argon2id for cross-platform consistency.
 */
export async function deriveMasterKey(passphrase: string, salt: string): Promise<ArrayBuffer> {
  const encoder = new TextEncoder();

  const keyMaterial = await crypto.subtle.importKey(
    'raw',
    encoder.encode(passphrase),
    'PBKDF2',
    false,
    ['deriveBits'],
  );

  const derivedBits = await crypto.subtle.deriveBits(
    {
      name: 'PBKDF2',
      salt: encoder.encode(salt),
      iterations: 600000,
      hash: 'SHA-256',
    },
    keyMaterial,
    256,
  );

  return derivedBits;
}

/** Attempt to refresh the access token using the refresh token. */
async function attemptTokenRefresh(): Promise<string | null> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return null;

  try {
    const res = await fetch(`${GATEWAY_URL}/v1/session/refresh`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ refresh_token: refreshToken }),
    });

    if (!res.ok) return null;

    const data = await res.json();
    if (data.access_token) {
      setSessionToken(data.access_token);
      if (data.refresh_token) {
        setRefreshToken(data.refresh_token);
      }
      return data.access_token;
    }
  } catch {
    // Network failure
  }
  return null;
}

/** Authenticated fetch with automatic 401 handling and token refresh. */
export async function authenticatedFetch(url: string, init?: RequestInit): Promise<Response> {
  const fullUrl = url.startsWith('http') ? url : `${GATEWAY_URL}${url}`;
  const token = getSessionToken();

  const response = await fetch(fullUrl, {
    ...init,
    headers: {
      ...authHeaders(),
      ...init?.headers,
    },
  });

  if (response.status === 401 && token) {
    const newToken = await attemptTokenRefresh();
    if (newToken) {
      // Retry with new token
      const retryHeaders = new Headers(init?.headers);
      retryHeaders.set('Authorization', `Bearer ${newToken}`);
      return fetch(fullUrl, {
        ...init,
        headers: retryHeaders,
      });
    }
    // Refresh failed — force logout
    await logout();
    // logout() redirects, so this won't really return
    return response;
  }

  return response;
}

/**
 * Make an authenticated GET request.
 */
export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const url = path.startsWith('http') ? path : `${GATEWAY_URL}${path}`;
  return authenticatedFetch(url, init);
}

/**
 * Get the current access token (alias for getSessionToken).
 * Used by the key management module to derive encryption keys.
 */
export function getAccessToken(): string | null {
  return getSessionToken();
}

/**
 * Check whether the user has a valid, non-expired session.
 */
export function isAuthenticated(): boolean {
  const token = getSessionToken();
  if (!token) return false;
  try {
    const parts = token.split('.');
    if (parts.length !== 3) return false;
    const payload = JSON.parse(atob(parts[1]));
    if (typeof payload.exp !== 'number') return true;
    return payload.exp > Date.now() / 1000;
  } catch {
    return false;
  }
}

/**
 * Make an authenticated POST request with a JSON body.
 */
export async function apiPost(path: string, body?: unknown): Promise<Response> {
  const url = path.startsWith('http') ? path : `${GATEWAY_URL}${path}`;
  return authenticatedFetch(url, {
    method: 'POST',
    headers: authHeaders(),
    body: body ? JSON.stringify(body) : undefined,
  });
}