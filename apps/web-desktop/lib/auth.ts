/**
 * Auth utilities for NomadOS web desktop client.
 *
 * Provides client-side key derivation for the vault service.
 * Phase 5 uses PBKDF2 (Web Crypto API) as a browser-compatible
 * alternative to Argon2id. Phase 6 will add WASM-based Argon2id.
 */

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

/**
 * Get the current session token from local storage.
 * Used as the Authorization header for authenticated API calls.
 */
export function getSessionToken(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('nomados-session');
}

/**
 * Derive a 256-bit master key from a passphrase and salt using PBKDF2.
 *
 * This is the Phase 5 browser-compatible alternative to Argon2id.
 * The same passphrase + salt must be used on every device to derive
 * the same master key. The salt is stored on the server (not secret;
 * its purpose is to be unique per user).
 *
 * Phase 6 will replace this with WASM-based Argon2id for consistency
 * with the server-side key derivation parameters.
 *
 * @param passphrase - User's passphrase (already normalized: trimmed, NFKC)
 * @param salt - Unique salt string per user (stored server-side)
 * @returns 32-byte (256-bit) master key as ArrayBuffer
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

  // PBKDF2 with 600,000 iterations as Phase 5 alternative to Argon2id.
  // Phase 6 will add WASM-based Argon2id for cross-platform consistency.
  const derivedBits = await crypto.subtle.deriveBits(
    {
      name: 'PBKDF2',
      salt: encoder.encode(salt),
      iterations: 600000,
      hash: 'SHA-256',
    },
    keyMaterial,
    256, // 32 bytes = 256 bits
  );

  return derivedBits;
}

/**
 * Make an authenticated POST request to the gateway.
 * Includes the session token as a Bearer authorization header.
 */
export async function apiPost(path: string, body: unknown): Promise<Response> {
  const token = getSessionToken();
  const headers: Record<string, string> = {
    'Content-Type': 'application/json',
  };
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return fetch(`${GATEWAY_URL}${path}`, {
    method: 'POST',
    headers,
    body: JSON.stringify(body),
  });
}

/**
 * Make an authenticated GET request to the gateway.
 * Includes the session token as a Bearer authorization header.
 */
export async function apiGet(path: string): Promise<Response> {
  const token = getSessionToken();
  const headers: Record<string, string> = {};
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }

  return fetch(`${GATEWAY_URL}${path}`, {
    method: 'GET',
    headers,
  });
}