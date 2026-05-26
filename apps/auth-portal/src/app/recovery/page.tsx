'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input, Card } from '@nomados/ui-components';

type RecoveryStep = 'identify' | 'verify' | 'reset';

export default function RecoveryPage() {
  const router = useRouter();
  const [step, setStep] = useState<RecoveryStep>('identify');
  const [identifier, setIdentifier] = useState('');
  const [recoveryKey, setRecoveryKey] = useState('');
  const [verificationCode, setVerificationCode] = useState('');
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [success, setSuccess] = useState(false);

  const handleIdentify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!identifier.trim()) {
      setError('Please enter your username or recovery key');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      // Phase 1: Mock - always succeed
      const res = await fetch(`${process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080'}/auth/recovery/begin`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ identifier: identifier.trim() }),
      });

      if (!res.ok) {
        // In Phase 1, the backend may not exist yet, fall back to mock
        if (res.status === 404) {
          setStep('verify');
          return;
        }
        throw new Error('Failed to start recovery');
      }

      setStep('verify');
    } catch (_e: unknown) {
      // Phase 1 mock: proceed to verify step anyway
      setStep('verify');
    } finally {
      setLoading(false);
    }
  };

  const handleVerify = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!verificationCode.trim()) {
      setError('Please enter the verification code');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080'}/auth/recovery/verify`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          identifier: identifier.trim(),
          code: verificationCode.trim(),
        }),
      });

      if (!res.ok) {
        if (res.status === 404) {
          // Phase 1 mock: proceed to reset step
          setStep('reset');
          return;
        }
        throw new Error('Verification failed');
      }

      setStep('reset');
    } catch (_e: unknown) {
      // Phase 1 mock: proceed to reset step anyway
      setStep('reset');
    } finally {
      setLoading(false);
    }
  };

  const handleReset = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!recoveryKey.trim()) {
      setError('Please enter your new recovery key or choose a new passkey');
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const res = await fetch(`${process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080'}/auth/recovery/reset`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          identifier: identifier.trim(),
          recoveryKey: recoveryKey.trim(),
        }),
      });

      if (!res.ok) {
        if (res.status === 404) {
          // Phase 1 mock: succeed
          setSuccess(true);
          return;
        }
        throw new Error('Recovery reset failed');
      }

      setSuccess(true);
    } catch (_e: unknown) {
      // Phase 1 mock: succeed anyway
      setSuccess(true);
    } finally {
      setLoading(false);
    }
  };

  if (success) {
    return (
      <div className="flex min-h-screen items-center justify-center p-4">
        <div className="w-full max-w-md">
          <Card className="border-nomados-border">
            <div className="flex flex-col items-center gap-6">
              <div className="flex h-14 w-14 items-center justify-center rounded-full bg-green-500/20">
                <svg
                  className="h-8 w-8 text-green-500"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M5 13l4 4L19 7"
                  />
                </svg>
              </div>
              <div className="text-center">
                <h1 className="text-2xl font-bold text-nomados-text">Recovery Complete</h1>
                <p className="mt-2 text-sm text-nomados-text-muted">
                  Your account has been recovered. You can now sign in with your new passkey.
                </p>
              </div>
              <Button
                variant="primary"
                size="lg"
                onClick={() => router.push('/login')}
                className="w-full"
              >
                Back to Login
              </Button>
            </div>
          </Card>
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <div className="w-full max-w-md">
        <Card className="border-nomados-border">
          <div className="flex flex-col items-center gap-6">
            {/* Header */}
            <div className="flex flex-col items-center gap-2">
              <div className="flex h-14 w-14 items-center justify-center rounded-xl bg-amber-500/20">
                <svg
                  className="h-8 w-8 text-amber-500"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={2}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"
                  />
                </svg>
              </div>
              <h1 className="text-2xl font-bold text-nomados-text">Account Recovery</h1>
              <p className="text-sm text-nomados-text-muted">
                {step === 'identify' && 'Enter your username or recovery key'}
                {step === 'verify' && 'Verify your identity'}
                {step === 'reset' && 'Set a new passkey'}
              </p>
            </div>

            {/* Step Indicators */}
            <div className="flex w-full items-center justify-center gap-2">
              {(['identify', 'verify', 'reset'] as RecoveryStep[]).map((s, i) => (
                <div key={s} className="flex items-center gap-2">
                  <div
                    className={`flex h-8 w-8 items-center justify-center rounded-full text-sm font-medium ${
                      step === s
                        ? 'bg-nomados-primary text-white'
                        : i < ['identify', 'verify', 'reset'].indexOf(step)
                        ? 'bg-green-500 text-white'
                        : 'bg-nomados-border text-nomados-text-muted'
                    }`}
                  >
                    {i + 1}
                  </div>
                  {i < 2 && (
                    <div className="h-0.5 w-8 bg-nomados-border" />
                  )}
                </div>
              ))}
            </div>

            {/* Step: Identify */}
            {step === 'identify' && (
              <form onSubmit={handleIdentify} className="w-full space-y-4">
                <Input
                  label="Username or Recovery Key"
                  placeholder="Enter your username or recovery key"
                  value={identifier}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    setIdentifier(e.target.value);
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
                  Continue
                </Button>
              </form>
            )}

            {/* Step: Verify */}
            {step === 'verify' && (
              <form onSubmit={handleVerify} className="w-full space-y-4">
                <div className="rounded-lg border border-nomados-border bg-nomados-background p-3">
                  <p className="text-sm text-nomados-text-muted">
                    In Phase 1, verification is simulated. Enter any code to proceed.
                  </p>
                </div>
                <Input
                  label="Verification Code"
                  placeholder="Enter the code sent to your device"
                  value={verificationCode}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    setVerificationCode(e.target.value);
                    setError(null);
                  }}
                  error={error || undefined}
                />
                <div className="flex gap-3">
                  <Button
                    type="button"
                    variant="secondary"
                    size="lg"
                    onClick={() => {
                      setStep('identify');
                      setError(null);
                    }}
                    className="flex-1"
                  >
                    Back
                  </Button>
                  <Button
                    type="submit"
                    variant="primary"
                    size="lg"
                    loading={loading}
                    className="flex-1"
                  >
                    Verify
                  </Button>
                </div>
              </form>
            )}

            {/* Step: Reset */}
            {step === 'reset' && (
              <form onSubmit={handleReset} className="w-full space-y-4">
                <div className="rounded-lg border border-nomados-border bg-nomados-background p-3">
                  <p className="text-sm text-nomados-text-muted">
                    In Phase 1, passkey registration is simulated. Enter any value to complete recovery.
                  </p>
                </div>
                <Input
                  label="New Recovery Key"
                  placeholder="Enter a new recovery key"
                  value={recoveryKey}
                  onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
                    setRecoveryKey(e.target.value);
                    setError(null);
                  }}
                  error={error || undefined}
                />
                <div className="flex gap-3">
                  <Button
                    type="button"
                    variant="secondary"
                    size="lg"
                    onClick={() => {
                      setStep('verify');
                      setError(null);
                    }}
                    className="flex-1"
                  >
                    Back
                  </Button>
                  <Button
                    type="submit"
                    variant="primary"
                    size="lg"
                    loading={loading}
                    className="flex-1"
                  >
                    Use Recovery Key
                  </Button>
                </div>
              </form>
            )}

            {/* Back to Login */}
            <a
              href="/login"
              className="text-sm text-nomados-primary hover:underline"
            >
              Back to Login
            </a>
          </div>
        </Card>
      </div>
    </div>
  );
}