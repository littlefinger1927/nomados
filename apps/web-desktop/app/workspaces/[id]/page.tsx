'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Button } from '@nomados/ui-components';

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

const MOCK_WORKSPACE: Workspace = {
  id: 'ws-demo',
  name: 'Demo Workspace',
  state: 'Running',
  createdAt: new Date(Date.now() - 86400000).toISOString(),
};

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

async function fetchWorkspaceFromApi(workspaceId: string): Promise<Workspace | null> {
  const token = localStorage.getItem('nomados-session');
  const res = await fetch(`${GATEWAY_URL}/workspace/${workspaceId}`, {
    headers: { Authorization: `Bearer ${token}` },
  });

  if (res.ok) {
    return await res.json();
  }
  return null;
}

export default function WorkspaceDetailPage() {
  const params = useParams();
  const router = useRouter();
  const workspaceId = params.id as string;

  const [workspace, setWorkspace] = useState<Workspace | null>(null);
  const [loading, setLoading] = useState(true);
  const [connecting, setConnecting] = useState(true);
  const [activeTab, setActiveTab] = useState<'files' | 'info'>('info');

  // Fetch workspace data on mount
  useEffect(() => {
    let cancelled = false;

    async function load() {
      try {
        const data = await fetchWorkspaceFromApi(workspaceId);
        if (!cancelled) {
          setWorkspace(data || { ...MOCK_WORKSPACE, id: workspaceId, name: `Workspace ${workspaceId}` });
        }
      } catch {
        if (!cancelled) {
          setWorkspace({ ...MOCK_WORKSPACE, id: workspaceId, name: `Workspace ${workspaceId}` });
        }
      } finally {
        if (!cancelled) {
          setLoading(false);
        }
      }
    }

    load();
    return () => { cancelled = true; };
  }, [workspaceId]);

  // Simulate connecting to stream when workspace is running
  useEffect(() => {
    if (workspace?.state === 'Running') {
      const timer = setTimeout(() => setConnecting(false), 2000);
      return () => clearTimeout(timer);
    }
  }, [workspace?.state]);

  const handleAction = async (action: 'pause' | 'stop' | 'resume' | 'start') => {
    try {
      const token = localStorage.getItem('nomados-session');
      await fetch(`${GATEWAY_URL}/workspace/${workspaceId}/${action}`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` },
      });
    } catch {
      // Mock - update locally
    }

    const stateMap: Record<string, WorkspaceState> = {
      start: 'Running',
      resume: 'Running',
      pause: 'Paused',
      stop: 'Stopped',
    };

    if (stateMap[action]) {
      setWorkspace((prev) =>
        prev ? { ...prev, state: stateMap[action] } : prev
      );
    }
  };

  const handleDelete = async () => {
    if (!confirm('Are you sure you want to delete this workspace? This action cannot be undone.')) return;

    try {
      const token = localStorage.getItem('nomados-session');
      await fetch(`${GATEWAY_URL}/workspace/${workspaceId}`, {
        method: 'DELETE',
        headers: { Authorization: `Bearer ${token}` },
      });
    } catch {
      // Ignore
    }

    router.push('/workspaces');
  };

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-nomados-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
      </div>
    );
  }

  if (!workspace) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-nomados-background">
        <div className="text-center">
          <h2 className="text-lg font-semibold text-nomados-text mb-2">Workspace not found</h2>
          <p className="text-sm text-nomados-text-muted mb-4">
            The workspace you are looking for does not exist.
          </p>
          <Button variant="primary" onClick={() => router.push('/workspaces')}>
            Back to Workspaces
          </Button>
        </div>
      </div>
    );
  }

  const isRunning = workspace.state === 'Running';
  const isPaused = workspace.state === 'Paused';
  const isStopped = workspace.state === 'Stopped';
  const isCreating = workspace.state === 'Creating';

  return (
    <div className="flex h-screen flex-col bg-nomados-background">
      {/* Header */}
      <header className="flex items-center justify-between border-b border-nomados-border bg-nomados-surface px-4 py-3">
        <div className="flex items-center gap-3">
          <Button variant="ghost" size="sm" onClick={() => router.push('/workspaces')}>
            <svg
              className="h-4 w-4 mr-1"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path strokeLinecap="round" strokeLinejoin="round" d="M15 19l-7-7 7-7" />
            </svg>
            Back
          </Button>
          <h1 className="text-lg font-semibold text-nomados-text">{workspace.name}</h1>
          <span
            className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${stateColors[workspace.state]}`}
          >
            {workspace.state}
          </span>
        </div>
        <div className="flex items-center gap-2">
          {isRunning && (
            <>
              <Button variant="secondary" size="sm" onClick={() => handleAction('pause')}>
                Pause
              </Button>
              <Button variant="secondary" size="sm" onClick={() => handleAction('stop')}>
                Stop
              </Button>
            </>
          )}
          {isPaused && (
            <Button variant="primary" size="sm" onClick={() => handleAction('resume')}>
              Resume
            </Button>
          )}
          {isStopped && (
            <Button variant="primary" size="sm" onClick={() => handleAction('start')}>
              Start
            </Button>
          )}
          <Button variant="danger" size="sm" onClick={handleDelete}>
            Delete
          </Button>
        </div>
      </header>

      {/* Main Content */}
      <div className="flex flex-1 overflow-hidden">
        {/* Stream Viewer Area */}
        <div className="flex-1 flex items-center justify-center bg-black relative">
          {isCreating && (
            <div className="flex flex-col items-center gap-3">
              <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
              <p className="text-sm text-nomados-text-muted">Creating workspace...</p>
            </div>
          )}
          {isRunning && connecting && (
            <div className="flex flex-col items-center gap-3">
              <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
              <p className="text-sm text-nomados-text-muted">Connecting to workspace...</p>
            </div>
          )}
          {isRunning && !connecting && (
            <div className="flex flex-col items-center gap-4 max-w-md text-center px-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-nomados-surface border border-nomados-border">
                <svg
                  className="h-8 w-8 text-nomados-primary"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-nomados-text">
                Stream Viewer
              </h3>
              <p className="text-sm text-nomados-text-muted leading-relaxed">
                The browser stream will connect via WebRTC when the
                streaming service is available. In Phase 1, this
                placeholder represents the remote browser viewport.
              </p>
              <div className="rounded-lg border border-dashed border-nomados-border px-4 py-3 text-xs text-nomados-text-muted">
                webrtc://workspace/{workspaceId}/stream
              </div>
            </div>
          )}
          {isPaused && (
            <div className="flex flex-col items-center gap-4 max-w-md text-center px-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-orange-500/10 border border-orange-500/20">
                <svg
                  className="h-8 w-8 text-orange-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-nomados-text">Workspace Paused</h3>
              <p className="text-sm text-nomados-text-muted">
                Resume this workspace to continue your session.
              </p>
            </div>
          )}
          {isStopped && (
            <div className="flex flex-col items-center gap-4 max-w-md text-center px-4">
              <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-red-500/10 border border-red-500/20">
                <svg
                  className="h-8 w-8 text-red-400"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z"
                  />
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z"
                  />
                </svg>
              </div>
              <h3 className="text-lg font-semibold text-nomados-text">Workspace Stopped</h3>
              <p className="text-sm text-nomados-text-muted">
                Start this workspace to begin a new session.
              </p>
            </div>
          )}
        </div>

        {/* Sidebar */}
        <aside className="w-72 border-l border-nomados-border bg-nomados-surface flex flex-col">
          {/* Tab Bar */}
          <div className="flex border-b border-nomados-border">
            <button
              onClick={() => setActiveTab('info')}
              className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
                activeTab === 'info'
                  ? 'text-nomados-primary border-b-2 border-nomados-primary'
                  : 'text-nomados-text-muted hover:text-nomados-text'
              }`}
            >
              Info
            </button>
            <button
              onClick={() => setActiveTab('files')}
              className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
                activeTab === 'files'
                  ? 'text-nomados-primary border-b-2 border-nomados-primary'
                  : 'text-nomados-text-muted hover:text-nomados-text'
              }`}
            >
              Files
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-4">
            {activeTab === 'info' && (
              <div className="space-y-4">
                <div>
                  <label className="text-xs font-medium text-nomados-text-muted uppercase tracking-wider">
                    Workspace ID
                  </label>
                  <p className="mt-1 text-sm text-nomados-text font-mono break-all">
                    {workspace.id}
                  </p>
                </div>
                <div>
                  <label className="text-xs font-medium text-nomados-text-muted uppercase tracking-wider">
                    State
                  </label>
                  <div className="mt-1">
                    <span
                      className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${stateColors[workspace.state]}`}
                    >
                      {workspace.state}
                    </span>
                  </div>
                </div>
                <div>
                  <label className="text-xs font-medium text-nomados-text-muted uppercase tracking-wider">
                    Created
                  </label>
                  <p className="mt-1 text-sm text-nomados-text">
                    {formatDate(workspace.createdAt)}
                  </p>
                </div>
                <div>
                  <label className="text-xs font-medium text-nomados-text-muted uppercase tracking-wider">
                    Connection
                  </label>
                  <p className="mt-1 text-sm text-nomados-text-muted">
                    {isRunning ? 'WebRTC (Phase 2)' : 'Disconnected'}
                  </p>
                </div>
              </div>
            )}

            {activeTab === 'files' && (
              <div className="flex flex-col items-center justify-center py-8 text-center">
                <svg
                  className="h-10 w-10 text-nomados-text-muted mb-3"
                  fill="none"
                  viewBox="0 0 24 24"
                  stroke="currentColor"
                  strokeWidth={1.5}
                >
                  <path
                    strokeLinecap="round"
                    strokeLinejoin="round"
                    d="M3 7v10a2 2 0 002 2h14a2 2 0 002-2V9a2 2 0 00-2-2h-6l-2-2H5a2 2 0 00-2 2z"
                  />
                </svg>
                <p className="text-sm text-nomados-text-muted">
                  Files will appear here
                </p>
                <p className="text-xs text-nomados-text-muted mt-1">
                  Connect to a running workspace to browse files
                </p>
              </div>
            )}
          </div>
        </aside>
      </div>
    </div>
  );
}