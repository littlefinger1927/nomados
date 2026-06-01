'use client';

import { useState, useEffect } from 'react';
import { useParams, useRouter } from 'next/navigation';
import { Button, WorkspaceDetailSkeleton } from '@nomados/ui-components';
import { StreamViewer } from './StreamViewer';

const GATEWAY_URL = process.env.NEXT_PUBLIC_GATEWAY_URL || 'http://localhost:8080';

type WorkspaceState = 'Creating' | 'Running' | 'Paused' | 'Stopped';

interface Workspace {
  id: string;
  name: string;
  state: WorkspaceState;
  createdAt: string;
  novncPort?: number;
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
  const res = await fetch(`${GATEWAY_URL}/v1/workspace/${workspaceId}`, {
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
  const [activeTab, setActiveTab] = useState<'files' | 'info'>('info');

  const isRunning = workspace?.state === 'Running';

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

  const handleAction = async (action: 'pause' | 'stop' | 'resume' | 'start') => {
    const actionEndpoints: Record<string, string> = {
      start: '/v1/workspace/start',
      resume: '/v1/workspace/resume',
      pause: '/v1/workspace/pause',
      stop: '/v1/workspace/stop',
    };

    try {
      const token = localStorage.getItem('nomados-session');
      await fetch(`${GATEWAY_URL}${actionEndpoints[action]}`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ id: { value: workspaceId } }),
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
      await fetch(`${GATEWAY_URL}/v1/workspace/destroy`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ id: { value: workspaceId } }),
      });
    } catch {
      // Ignore
    }

    router.push('/workspaces');
  };

  if (loading) {
    return <WorkspaceDetailSkeleton />;
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
        <div className="flex-1 flex items-center justify-center bg-black relative md:pb-0 pb-14">
          <StreamViewer
            workspaceId={workspaceId}
            workspaceState={workspace.state}
            novncPort={workspace.novncPort}
          />
        </div>

        {/* Sidebar - hidden on mobile, narrower on tablet, full on desktop */}
        <aside className="hidden md:flex w-64 lg:w-72 border-l border-nomados-border bg-nomados-surface flex-col">
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
                    {isRunning ? 'Active' : 'Disconnected'}
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

      {/* Mobile bottom tab bar for Info/Files switching */}
      <nav className="md:hidden fixed bottom-0 inset-x-0 z-30 border-t border-nomados-border bg-nomados-surface">
        <div className="flex">
          <button
            onClick={() => setActiveTab('info')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'info'
                ? 'text-nomados-primary border-t-2 border-t-nomados-primary'
                : 'text-nomados-text-muted'
            }`}
          >
            Info
          </button>
          <button
            onClick={() => setActiveTab('files')}
            className={`flex-1 px-4 py-3 text-sm font-medium transition-colors ${
              activeTab === 'files'
                ? 'text-nomados-primary border-t-2 border-t-nomados-primary'
                : 'text-nomados-text-muted'
            }`}
          >
            Files
          </button>
        </div>
      </nav>

      {/* Mobile panel overlay for Info/Files content */}
      {activeTab === 'info' && (
        <div className="md:hidden fixed inset-x-0 bottom-12 z-20 max-h-[50vh] overflow-y-auto border-t border-nomados-border bg-nomados-surface p-4">
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
                {isRunning ? 'Active' : 'Disconnected'}
              </p>
            </div>
          </div>
        </div>
      )}
      {activeTab === 'files' && (
        <div className="md:hidden fixed inset-x-0 bottom-12 z-20 max-h-[50vh] overflow-y-auto border-t border-nomados-border bg-nomados-surface p-4">
          <div className="flex flex-col items-center justify-center py-4 text-center">
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
        </div>
      )}
    </div>
  );
}