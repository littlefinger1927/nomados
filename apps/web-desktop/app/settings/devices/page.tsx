'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button, ErrorBoundary } from '@nomados/ui-components';

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

interface CredentialInfo {
  device_id: { value: string };
  device_name: string;
  attestation_type: string;
  created_at: number;
  last_seen: number;
}

export default function DevicesPage() {
  const router = useRouter();
  const [credentials, setCredentials] = useState<CredentialInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [adding, setAdding] = useState(false);

  async function fetchCredentials() {
    try {
      const token = localStorage.getItem('nomados-session');
      if (!token) {
        router.push('/login');
        return;
      }

      const payload = JSON.parse(atob(token.split('.')[1]));
      const userId = payload.sub;

      const res = await fetch(`${GATEWAY_URL}/v1/auth/list_credentials`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ user_id: { value: userId } }),
      });

      if (res.ok) {
        const data = await res.json();
        setCredentials(data.credentials || []);
      } else {
        // In development, show empty state
        setCredentials([]);
      }
    } catch {
      setCredentials([]);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    fetchCredentials();
  }, []);

  async function handleAddCredential() {
    setAdding(true);
    setError(null);

    try {
      const token = localStorage.getItem('nomados-session');
      if (!token) {
        router.push('/login');
        return;
      }

      const payload = JSON.parse(atob(token.split('.')[1]));
      const userId = payload.sub;

      // Step 1: Request a new credential challenge from the server
      const res = await fetch(`${GATEWAY_URL}/v1/auth/add_credential`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          user_id: { value: userId },
          device_name: `Passkey ${new Date().toLocaleDateString()}`,
        }),
      });

      if (!res.ok) {
        throw new Error('Failed to initiate credential registration');
      }

      const data = await res.json();

      // Step 2: Use WebAuthn API to create a new credential
      if (navigator.credentials && 'create' in navigator.credentials) {
        const challenge = new Uint8Array(data.webauthn_challenge);
        const publicKeyCredential = await navigator.credentials.create({
          publicKey: {
            challenge,
            rp: {
              name: 'NomadOS',
              id: window.location.hostname,
            },
            user: {
              id: new Uint8Array(16),
              name: payload.sub || 'user',
              displayName: 'User',
            },
            pubKeyCredParams: [
              { type: 'public-key', alg: -7 },
              { type: 'public-key', alg: -257 },
            ],
            authenticatorSelection: {
              authenticatorAttachment: 'cross-platform',
              userVerification: 'preferred',
            },
          },
        });

        if (publicKeyCredential) {
          const pkCred = publicKeyCredential as PublicKeyCredential;
          const response = pkCred.response as AuthenticatorAttestationResponse;
          // Step 3: Verify the credential with the server
          const verifyRes = await fetch(`${GATEWAY_URL}/v1/auth/register_verify`, {
            method: 'POST',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${token}`,
            },
            body: JSON.stringify({
              credential_response: Array.from(new Uint8Array(response.clientDataJSON)),
              device_signature: [],
              user_id: { value: userId },
              device_id: data.device_id,
            }),
          });

          if (!verifyRes.ok) {
            throw new Error('Failed to verify new credential');
          }

          await fetchCredentials();
        }
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to add credential');
    } finally {
      setAdding(false);
    }
  }

  async function handleRemoveCredential(deviceId: string) {
    if (!confirm('Remove this passkey? You must have at least one passkey remaining.')) return;

    try {
      const token = localStorage.getItem('nomados-session');
      if (!token) return;

      const payload = JSON.parse(atob(token.split('.')[1]));
      const userId = payload.sub;

      const res = await fetch(`${GATEWAY_URL}/v1/auth/remove_credential`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({
          user_id: { value: userId },
          device_id: { value: deviceId },
        }),
      });

      if (res.ok) {
        await fetchCredentials();
      } else {
        const data = await res.json();
        setError(data.message || 'Failed to remove credential');
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to remove credential');
    }
  }

  function formatDate(timestamp: number): string {
    if (!timestamp) return 'Never';
    return new Date(timestamp * 1000).toLocaleDateString('en-US', {
      month: 'short',
      day: 'numeric',
      year: 'numeric',
    });
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-nomados-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
      </div>
    );
  }

  return (
    <div className="flex min-h-screen flex-col bg-nomados-background">
      <header className="border-b border-nomados-border bg-nomados-surface px-6 py-4">
        <div className="mx-auto max-w-3xl flex items-center justify-between">
          <div>
            <h1 className="text-lg font-semibold text-nomados-text">Devices & Security</h1>
            <p className="text-sm text-nomados-text-muted mt-1">
              Manage your passkeys and security credentials
            </p>
          </div>
          <Button variant="primary" size="sm" onClick={handleAddCredential} disabled={adding}>
            {adding ? 'Adding...' : 'Add Passkey'}
          </Button>
        </div>
      </header>

      <main className="flex-1 px-6 py-8">
        <div className="mx-auto max-w-3xl space-y-4">
          {error && (
            <div className="rounded-md bg-red-500/10 border border-red-500/20 px-4 py-3 text-sm text-red-400">
              {error}
              <button
                onClick={() => setError(null)}
                className="ml-2 text-red-400/70 hover:text-red-400"
              >
                Dismiss
              </button>
            </div>
          )}

          {credentials.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-12 text-center">
              <svg
                className="h-12 w-12 text-nomados-text-muted mb-4"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={1.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.639.403l-1.832 1.38a4.5 4.5 0 01-6.364-6.364l1.38-1.832c.377-.48.5-1.076.403-1.639A6 6 0 1121.75 8.25z"
                />
              </svg>
              <h3 className="text-lg font-semibold text-nomados-text mb-1">No passkeys yet</h3>
              <p className="text-sm text-nomados-text-muted mb-4">
                Add a passkey to sign in without passwords
              </p>
              <Button variant="primary" onClick={handleAddCredential} disabled={adding}>
                {adding ? 'Adding...' : 'Add Your First Passkey'}
              </Button>
            </div>
          ) : (
            <div className="space-y-3">
              {credentials.map((cred) => (
                <div
                  key={cred.device_id.value}
                  className="rounded-lg border border-nomados-border bg-nomados-surface p-4 flex items-center justify-between"
                >
                  <div className="flex items-center gap-3">
                    <div className="flex h-10 w-10 items-center justify-center rounded-full bg-nomados-primary/10">
                      <svg
                        className="h-5 w-5 text-nomados-primary"
                        fill="none"
                        viewBox="0 0 24 24"
                        stroke="currentColor"
                        strokeWidth={1.5}
                      >
                        <path
                          strokeLinecap="round"
                          strokeLinejoin="round"
                          d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.639.403l-1.832 1.38a4.5 4.5 0 01-6.364-6.364l1.38-1.832c.377-.48.5-1.076.403-1.639A6 6 0 1121.75 8.25z"
                        />
                      </svg>
                    </div>
                    <div>
                      <p className="text-sm font-medium text-nomados-text">
                        {cred.device_name || 'Unnamed Passkey'}
                      </p>
                      <p className="text-xs text-nomados-text-muted">
                        Added {formatDate(cred.created_at)}
                        {cred.last_seen > 0 && ` · Last used ${formatDate(cred.last_seen)}`}
                      </p>
                    </div>
                  </div>
                  <Button
                    variant="danger"
                    size="sm"
                    onClick={() => handleRemoveCredential(cred.device_id.value)}
                    disabled={credentials.length <= 1}
                  >
                    Remove
                  </Button>
                </div>
              ))}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}