'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input, Card } from '@nomados/ui-components';

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

// Decode a base64url string to ArrayBuffer
function base64urlDecode(str: string): ArrayBuffer {
  // Base64url to Base64
  let base64 = str.replace(/-/g, '+').replace(/_/g, '/');
  // Pad with =
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

// Encode an ArrayBuffer to base64url string
function base64urlEncode(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer);
  let binary = '';
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]);
  }
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '');
}

type AuthMode = 'login' | 'register';

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [mode, setMode] = useState<AuthMode>('login');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) {
      setError('Please enter a username');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      if (mode === 'register') {
        await handleRegister();
      } else {
        await handleLogin();
      }
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Authentication failed';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  const handleRegister = async () => {
    // Step 1: Get registration challenge from server
    const beginRes = await fetch(`${GATEWAY_URL}/auth/register/begin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.trim() }),
    });

    if (!beginRes.ok) {
      // In Phase 1, the backend may not exist yet, fall back to mock
      if (beginRes.status === 404) {
        handleMockAuth();
        return;
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
    const finishRes = await fetch(`${GATEWAY_URL}/auth/register/finish`, {
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
    localStorage.setItem('nomados-session', data.token);
    router.push('/workspaces');
  };

  const handleLogin = async () => {
    // Step 1: Get authentication challenge from server
    const beginRes = await fetch(`${GATEWAY_URL}/auth/login/begin`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ username: username.trim() }),
    });

    if (!beginRes.ok) {
      // In Phase 1, the backend may not exist yet, fall back to mock
      if (beginRes.status === 404) {
        handleMockAuth();
        return;
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
    const finishRes = await fetch(`${GATEWAY_URL}/auth/login/finish`, {
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
    localStorage.setItem('nomados-session', data.token);
    router.push('/workspaces');
  };

  const handleMockAuth = () => {
    // Phase 1 fallback: store mock session and redirect
    const mockToken = btoa(JSON.stringify({
      sub: username.trim(),
      iat: Date.now(),
      exp: Date.now() + 86400000,
    }));
    localStorage.setItem('nomados-session', mockToken);
    router.push('/workspaces');
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md">
        <Card className="border-nomados-border">
          <div className="flex flex-col items-center gap-6">
            {/* Logo / Title */}
            <div className="flex flex-col items-center gap-2">
              <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-nomados-primary">
                <svg
                  className="h-8 w-8 text-white"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2z"
                  />
                </svg>
              </div>
              <h1 className="text-2xl font-bold text-nomados-text">NomadOS</h1>
              <p className="text-sm text-nomados-text-muted">Sovereign Compute Platform</p>
            </div>

            {/* Auth Form */}
            <form onSubmit={handleSubmit} className="w-full space-y-4">
              <Input
                label="Username"
                placeholder="Enter your username"
                value={username}
                onChange={(e) => {
                  setUsername(e.target.value);
                  setError(null);
                }}
                error={error || undefined}
              />

              <Button
                type="submit"
                variant="primary"
                size="lg"
                loading={loading}
                className="w-full"
              >
                {mode === 'register' ? 'Create Account with Passkey' : 'Sign in with Passkey'}
              </Button>
            </form>

            {/* Toggle Mode */}
            <p className="text-sm text-nomados-text-muted">
              {mode === 'login' ? (
                <>
                  New here?{' '}
                  <button
                    type="button"
                    onClick={() => {
                      setMode('register');
                      setError(null);
                    }}
                    className="text-nomados-primary hover:underline"
                  >
                    Create an account
                  </button>
                </>
              ) : (
                <>
                  Already have an account?{' '}
                  <button
                    type="button"
                    onClick={() => {
                      setMode('login');
                      setError(null);
                    }}
                    className="text-nomados-primary hover:underline"
                  >
                    Sign in
                  </button>
                </>
              )}
            </p>

            {/* Phase 1 Notice */}
            <p className="text-xs text-nomados-text-muted text-center leading-relaxed">
              Passkey authentication requires a running gateway service.
              <br />
              In Phase 1, a mock session is used when the gateway is unavailable.
            </p>
          </div>
        </Card>
      </div>
    </div>
  );
}