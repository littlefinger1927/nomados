'use client';

import { useState, useEffect } from 'react';
import { useRouter } from 'next/navigation';
import { Button, Input, Card, Modal } from '@nomados/ui-components';

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

type WorkspaceState = 'Creating' | 'Running' | 'Paused' | 'Stopped';

interface Workspace {
  id: string;
  name: string;
  state: WorkspaceState;
  createdAt: string;
}

const stateColors: Record<WorkspaceState, string> = {
  Creating: 'bg-yellow-500/20 text-yellow-400',
  Running: 'bg-green-500/20 text-green-400',
  Paused: 'bg-orange-500/20 text-orange-400',
  Stopped: 'bg-red-500/20 text-red-400',
};

// Mock data for Phase 1 when gateway is not available
const MOCK_WORKSPACES: Workspace[] = [
  {
    id: 'ws-1',
    name: 'Development',
    state: 'Running',
    createdAt: new Date(Date.now() - 86400000 * 3).toISOString(),
  },
  {
    id: 'ws-2',
    name: 'Staging',
    state: 'Paused',
    createdAt: new Date(Date.now() - 86400000 * 7).toISOString(),
  },
  {
    id: 'ws-3',
    name: 'Testing',
    state: 'Stopped',
    createdAt: new Date(Date.now() - 86400000 * 14).toISOString(),
  },
];

function getUserInitial(): string {
  if (typeof window === 'undefined') return '?';
  try {
    const token = localStorage.getItem('nomados-session');
    if (token) {
      const payload = JSON.parse(atob(token));
      return (payload.sub || '?')[0].toUpperCase();
    }
  } catch {
    // ignore
  }
  return '?';
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  });
}

async function fetchWorkspacesFromApi(): Promise<Workspace[]> {
  const token = localStorage.getItem('nomados-session');
  const res = await fetch(`${GATEWAY_URL}/workspace/list`, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (res.ok) {
    const data = await res.json();
    return data.workspaces || data || [];
  }
  return MOCK_WORKSPACES;
}

export default function WorkspacesPage() {
  const router = useRouter();
  const [workspaces, setWorkspaces] = useState<Workspace[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreateModal, setShowCreateModal] = useState(false);
  const [newWorkspaceName, setNewWorkspaceName] = useState('');
  const [creating, setCreating] = useState(false);
  const [userInitial, setUserInitial] = useState('?');

  // Load user initial on mount
  useEffect(() => {
    setUserInitial(getUserInitial());
  }, []);

  // Fetch workspaces on mount
  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const data = await fetchWorkspacesFromApi();
        if (!cancelled) {
          setWorkspaces(data);
        }
      } catch {
        if (!cancelled) {
          setWorkspaces(MOCK_WORKSPACES);
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    load();
    return () => { cancelled = true; };
  }, []);

  const handleCreateWorkspace = async () => {
    if (!newWorkspaceName.trim()) return;

    setCreating(true);
    try {
      const token = localStorage.getItem('nomados-session');
      const res = await fetch(`${GATEWAY_URL}/workspace/create`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ name: newWorkspaceName.trim() }),
      });

      if (res.ok) {
        const ws = await res.json();
        setWorkspaces((prev) => [...prev, ws]);
      } else {
        // Mock: add workspace locally
        addMockWorkspace(newWorkspaceName.trim());
      }
    } catch {
      addMockWorkspace(newWorkspaceName.trim());
    } finally {
      setCreating(false);
      setNewWorkspaceName('');
      setShowCreateModal(false);
    }
  };

  const addMockWorkspace = (name: string) => {
    const newWs: Workspace = {
      id: `ws-${Date.now()}`,
      name,
      state: 'Creating',
      createdAt: new Date().toISOString(),
    };
    setWorkspaces((prev) => [...prev, newWs]);

    // Simulate transition to Running after a delay
    setTimeout(() => {
      setWorkspaces((prev) =>
        prev.map((w) =>
          w.id === newWs.id ? { ...w, state: 'Running' as WorkspaceState } : w
        )
      );
    }, 2000);
  };

  const handleDeleteWorkspace = async (id: string) => {
    if (!confirm('Are you sure you want to delete this workspace?')) return;

    try {
      const token = localStorage.getItem('nomados-session');
      await fetch(`${GATEWAY_URL}/workspace/${id}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
    } catch {
      // Ignore - remove locally anyway for Phase 1
    }

    setWorkspaces((prev) => prev.filter((w) => w.id !== id));
  };

  const handleAction = async (id: string, action: 'connect' | 'resume' | 'pause' | 'stop') => {
    const actionMap: Record<string, string> = {
      connect: '/connect',
      resume: '/resume',
      pause: '/pause',
      stop: '/stop',
    };

    try {
      const token = localStorage.getItem('nomados-session');
      await fetch(`${GATEWAY_URL}/workspace/${id}${actionMap[action]}`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
    } catch {
      // Mock action in Phase 1
    }

    if (action === 'connect') {
      router.push(`/workspaces/${id}`);
      return;
    }

    const stateMap: Record<string, WorkspaceState> = {
      resume: 'Running',
      pause: 'Paused',
      stop: 'Stopped',
    };

    if (stateMap[action]) {
      setWorkspaces((prev) =>
        prev.map((w) => (w.id === id ? { ...w, state: stateMap[action] } : w))
      );
    }
  };

  return (
    <div className="min-h-screen bg-nomados-background">
      {/* Header */}
      <header className="border-b border-nomados-border bg-nomados-surface">
        <div className="mx-auto flex max-w-7xl items-center justify-between px-6 py-4">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 items-center justify-center rounded-lg bg-nomados-primary">
              <svg
                className="h-5 w-5 text-white"
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
            <h1 className="text-lg font-bold text-nomados-text">NomadOS</h1>
          </div>
          <div className="flex items-center gap-3">
            <Button
              variant="primary"
              size="sm"
              onClick={() => setShowCreateModal(true)}
            >
              Create Workspace
            </Button>
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-nomados-primary text-sm font-medium text-white">
              {userInitial}
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="mx-auto max-w-7xl px-6 py-8">
        <div className="mb-6">
          <h2 className="text-xl font-semibold text-nomados-text">Workspaces</h2>
          <p className="text-sm text-nomados-text-muted">
            Manage your isolated compute environments
          </p>
        </div>

        {loading ? (
          <div className="flex items-center justify-center py-20">
            <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
          </div>
        ) : workspaces.length === 0 ? (
          /* Empty State */
          <div className="flex flex-col items-center justify-center py-20">
            <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-nomados-surface border border-nomados-border mb-4">
              <svg
                className="h-8 w-8 text-nomados-text-muted"
                fill="none"
                viewBox="0 0 24 24"
                stroke="currentColor"
                strokeWidth={1.5}
              >
                <path
                  strokeLinecap="round"
                  strokeLinejoin="round"
                  d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
                />
              </svg>
            </div>
            <h3 className="text-lg font-medium text-nomados-text mb-1">No workspaces yet</h3>
            <p className="text-sm text-nomados-text-muted mb-4">
              Create your first workspace to get started.
            </p>
            <Button variant="primary" onClick={() => setShowCreateModal(true)}>
              Create Workspace
            </Button>
          </div>
        ) : (
          /* Workspace Grid */
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {workspaces.map((ws) => (
              <Card key={ws.id} className="hover:border-nomados-primary/50 transition-colors">
                <div className="flex flex-col gap-3">
                  <div className="flex items-start justify-between">
                    <h3 className="font-semibold text-nomados-text">{ws.name}</h3>
                    <span
                      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${stateColors[ws.state]}`}
                    >
                      {ws.state}
                    </span>
                  </div>

                  <p className="text-xs text-nomados-text-muted">
                    Created {formatDate(ws.createdAt)}
                  </p>

                  <div className="flex items-center gap-2 pt-2 border-t border-nomados-border">
                    {ws.state === 'Running' && (
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() => handleAction(ws.id, 'connect')}
                      >
                        Connect
                      </Button>
                    )}
                    {ws.state === 'Paused' && (
                      <Button
                        variant="primary"
                        size="sm"
                        onClick={() => handleAction(ws.id, 'resume')}
                      >
                        Resume
                      </Button>
                    )}
                    {ws.state === 'Running' && (
                      <Button
                        variant="secondary"
                        size="sm"
                        onClick={() => handleAction(ws.id, 'pause')}
                      >
                        Pause
                      </Button>
                    )}
                    {ws.state === 'Running' && (
                      <Button
                        variant="ghost"
                        size="sm"
                        onClick={() => handleAction(ws.id, 'stop')}
                      >
                        Stop
                      </Button>
                    )}
                    <div className="flex-1" />
                    <Button
                      variant="danger"
                      size="sm"
                      onClick={() => handleDeleteWorkspace(ws.id)}
                    >
                      Delete
                    </Button>
                  </div>
                </div>
              </Card>
            ))}
          </div>
        )}
      </main>

      {/* Create Workspace Modal */}
      <Modal
        isOpen={showCreateModal}
        onClose={() => {
          setShowCreateModal(false);
          setNewWorkspaceName('');
        }}
        title="Create Workspace"
      >
        <form
          onSubmit={(e) => {
            e.preventDefault();
            handleCreateWorkspace();
          }}
          className="space-y-4"
        >
          <Input
            label="Workspace Name"
            placeholder="e.g. Development"
            value={newWorkspaceName}
            onChange={(e) => setNewWorkspaceName(e.target.value)}
          />
          <div className="flex justify-end gap-2">
            <Button
              variant="secondary"
              onClick={() => {
                setShowCreateModal(false);
                setNewWorkspaceName('');
              }}
            >
              Cancel
            </Button>
            <Button
              variant="primary"
              type="submit"
              loading={creating}
              disabled={!newWorkspaceName.trim()}
            >
              Create
            </Button>
          </div>
        </form>
      </Modal>
    </div>
  );
}