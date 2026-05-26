'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input, Card } from '@nomados/ui-components';
import { isWebAuthnSupported, authenticatePasskey, registerPasskey, storeSession } from '@/lib/webauthn';

type AuthMode = 'login' | 'register';

export default function LoginPage() {
  const router = useRouter();
  const [username, setUsername] = useState('');
  const [mode, setMode] = useState<AuthMode>('login');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [webAuthnSupported, setWebAuthnSupported] = useState(true);

  useEffect(() => {
    setWebAuthnSupported(isWebAuthnSupported());
  }, []);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!username.trim()) {
      setError('Please enter a username');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      let result;
      if (mode === 'register') {
        result = await registerPasskey(username.trim());
      } else {
        result = await authenticatePasskey(username.trim());
      }

      if (result.success && result.token) {
        storeSession(result.token);
        // Redirect to the web-desktop app
        window.location.href = process.env.NEXT_PUBLIC_DESKTOP_URL || '/';
      } else {
        setError(result.error || 'Authentication failed');
      }
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : 'Authentication failed';
      setError(message);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md">
        <Card className="border-nomados-border">
          <div className="flex flex-col items-center gap-6">
            {/* Logo / Shield */}
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
                    d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"
                  />
                </svg>
              </div>
              <h1 className="text-2xl font-bold text-nomados-text">Sign In</h1>
              <p className="text-sm text-nomados-text-muted">NomadOS Authentication Portal</p>
            </div>

            {/* WebAuthn Not Supported Warning */}
            {!webAuthnSupported && (
              <div className="w-full rounded-lg border border-nomados-danger bg-nomados-danger/10 px-4 py-3 text-sm text-nomados-danger">
                Your browser does not support WebAuthn passkeys. Please use a modern browser with biometric or security key support.
              </div>
            )}

            {/* Auth Form */}
            <form onSubmit={handleSubmit} className="w-full space-y-4">
              <Input
                label="Username"
                placeholder="Enter your username"
                value={username}
                onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                  setUsername(e.target.value);
                  setError(null);
                }}
                error={error || undefined}
                disabled={!webAuthnSupported}
              />

              <Button
                type="submit"
                variant="primary"
                size="lg"
                loading={loading}
                disabled={!webAuthnSupported}
                className="w-full"
              >
                {mode === 'register' ? 'Create Account with Passkey' : 'Sign in with Passkey'}
              </Button>
            </form>

            {/* Toggle Mode */}
            <p className="text-sm text-nomados-text-muted">
              {mode === 'login' ? (
                <>
                  Don&apos;t have a passkey?{' '}
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

            {/* Recovery Link */}
            {mode === 'login' && (
              <p className="text-sm text-nomados-text-muted">
                Lost your device?{' '}
                <a
                  href="/recovery"
                  className="text-nomados-primary hover:underline"
                >
                  Recover your account
                </a>
              </p>
            )}

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