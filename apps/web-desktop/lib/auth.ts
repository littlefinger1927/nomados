/**
 * Auth utilities for NomadOS web desktop client.
 *
 * Provides client-side key derivation for the vault service and
 * authenticated API request helpers.
 */

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

/**
 * Get the current session token from local storage.
 */
export function getSessionToken(): string | null {
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