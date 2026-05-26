/**
 * WebAuthn utility functions for passkey authentication.
 * Handles base64url encoding/decoding and WebAuthn API interactions.
 */

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

/**
 * Check if WebAuthn is supported in the current browser.
 */
export function isWebAuthnSupported(): boolean {
  return typeof window !== 'undefined' &&
    !!window.PublicKeyCredential &&
    typeof navigator.credentials !== 'undefined';
}

/**
 * Decode a base64url string to ArrayBuffer.
 */
export function base64urlDecode(str: string): ArrayBuffer {
  let base64 = str.replace(/-/g, '+').replace(/_/g, '/');
  while (base64.length % 4 !== 0) {
    base64 += '=';
  }
  const binary = atob(base64);
  const bytes = new Uint8Array(binary.length);
  for (let i = 0; i < binary.length; i++) {
    bytes[i] = binary.charCodeAt(i);
  }
  return bytes.buffer;
}

/**
 * Encode an ArrayBuffer to base64url string.
 */
export function base64urlEncode(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

export interface AuthResult {
  success: boolean;
  token?: string;
  error?: string;
}

/**
 * Start a WebAuthn registration flow.
 * Creates a new passkey credential for the given username.
 */
export async function registerPasskey(username: string): Promise<AuthResult> {
  try {
    // Step 1: Get registration challenge from server
    const beginRes = await fetch(`${GATEWAY_URL}/v1/auth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username }),
    });

    if (!beginRes.ok) {
      if (beginRes.status === 404) {
        return mockAuth(username);
      }
      throw new Error('Failed to start registration');
    }

    const options = await beginRes.json();

    // Step 2: Create WebAuthn credential
    const publicKey: PublicKeyCredentialCreationOptions = {
      ...options,
      challenge: base64urlDecode(options.challenge),
      user: {
        ...options.user,
        id: base64urlDecode(options.user.id),
      },
      excludeCredentials: (options.excludeCredentials || []).map(
        (cred: { id: string; type: PublicKeyCredentialType; transports?: AuthenticatorTransport[] }) => ({
          ...cred,
          id: base64urlDecode(cred.id),
        })
      ),
    };

    const credential = await navigator.credentials.create({ publicKey }) as PublicKeyCredential | null;
    if (!credential) throw new Error('Failed to create credential');

    const response = credential.response as AuthenticatorAttestationResponse;

    // Step 3: Send credential to server
    const finishRes = await fetch(`${GATEWAY_URL}/v1/auth/register_verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: credential.id,
        rawId: base64urlEncode(credential.rawId),
        type: credential.type,
        response: {
          attestationObject: base64urlEncode(response.attestationObject),
          clientDataJSON: base64urlEncode(response.clientDataJSON),
        },
      }),
    });

    if (!finishRes.ok) throw new Error('Registration failed');

    const data = await finishRes.json();
    return { success: true, token: data.token };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : 'Registration failed',
    };
  }
}

/**
 * Start a WebAuthn authentication flow.
 * Authenticates with an existing passkey for the given username.
 */
export async function authenticatePasskey(username: string): Promise<AuthResult> {
  try {
    // Step 1: Get authentication challenge from server
    const beginRes = await fetch(`${GATEWAY_URL}/v1/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username }),
    });

    if (!beginRes.ok) {
      if (beginRes.status === 404) {
        return mockAuth(username);
      }
      throw new Error('Failed to start login');
    }

    const options = await beginRes.json();

    // Step 2: Get WebAuthn assertion
    const publicKey: PublicKeyCredentialRequestOptions = {
      ...options,
      challenge: base64urlDecode(options.challenge),
      allowCredentials: (options.allowCredentials || []).map(
        (cred: { id: string; type: PublicKeyCredentialType; transports?: AuthenticatorTransport[] }) => ({
          ...cred,
          id: base64urlDecode(cred.id),
        })
      ),
    };

    const assertion = await navigator.credentials.get({ publicKey }) as PublicKeyCredential | null;
    if (!assertion) throw new Error('Authentication cancelled');

    const assertionResponse = assertion.response as AuthenticatorAssertionResponse;

    // Step 3: Send assertion to server
    const finishRes = await fetch(`${GATEWAY_URL}/v1/auth/login_verify`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        id: assertion.id,
        rawId: base64urlEncode(assertion.rawId),
        type: assertion.type,
        response: {
          authenticatorData: base64urlEncode(assertionResponse.authenticatorData),
          clientDataJSON: base64urlEncode(assertionResponse.clientDataJSON),
          signature: base64urlEncode(assertionResponse.signature),
          userHandle: assertionResponse.userHandle
            ? base64urlEncode(assertionResponse.userHandle)
            : null,
        },
      }),
    });

    if (!finishRes.ok) throw new Error('Login failed');

    const data = await finishRes.json();
    return { success: true, token: data.token };
  } catch (err) {
    return {
      success: false,
      error: err instanceof Error ? err.message : 'Authentication failed',
    };
  }
}

/**
 * Phase 1 fallback: create a mock session token when gateway is unavailable.
 */
function mockAuth(username: string): AuthResult {
  const mockToken = btoa(JSON.stringify({
    sub: username,
    iat: Date.now(),
    exp: Date.now() + 86400000,
  }));
  return { success: true, token: mockToken };
}

/**
 * Store session token in localStorage.
 */
export function storeSession(token: string): void {
  localStorage.setItem('nomados-session', token);
}

/**
 * Retrieve session token from localStorage.
 */
export function getSession(): string | null {
  if (typeof window === 'undefined') return null;
  return localStorage.getItem('nomados-session');
}

/**
 * Clear session token from localStorage.
 */
export function clearSession(): void {
  localStorage.removeItem('nomados-session');
}

/**
 * Check if there is an active session.
 */
export function isAuthenticated(): boolean {
  const token = getSession();
  if (!token) return false;
  try {
    const payload = JSON.parse(atob(token));
    return payload.exp > Date.now();
  } catch {
    return false;
  }
}