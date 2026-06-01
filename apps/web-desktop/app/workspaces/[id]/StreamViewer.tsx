'use client';

import { useState, useEffect, useCallback } from 'react';
import {
  WorkspaceConnection,
  ConnectionState,
  isWebRTCAvailable,
} from '@/lib/webrtc';

interface StreamViewerProps {
  workspaceId: string;
  workspaceState: 'Creating' | 'Running' | 'Paused' | 'Stopped';
  novncPort?: number;
}

export function StreamViewer({ workspaceId, workspaceState, novncPort }: StreamViewerProps) {
  const [webrtcState, setWebrtcState] = useState<ConnectionState>('disconnected');
  const [connection, setConnection] = useState<WorkspaceConnection | null>(null);
  const webrtcAvailable = isWebRTCAvailable();
  const isRunning = workspaceState === 'Running';

  const connectWebRTC = useCallback(async () => {
    if (connection) return;

    const conn = new WorkspaceConnection({
      workspaceId,
      onStateChange: setWebrtcState,
    });

    setConnection(conn);
    try {
      await conn.connect();
    } catch (err) {
      console.error('WebRTC connection failed, falling back to noVNC:', err);
    }
  }, [workspaceId, connection]);

  useEffect(() => {
    if (isRunning && webrtcAvailable && !connection) {
      connectWebRTC();
    }
  }, [isRunning, webrtcAvailable, connection, connectWebRTC]);

  useEffect(() => {
    return () => {
      connection?.disconnect();
    };
  }, [connection]);

  // Building states
  if (workspaceState === 'Creating') {
    return (
      <div className="flex flex-col items-center gap-3">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
        <p className="text-sm text-nomados-text-muted">Creating workspace...</p>
      </div>
    );
  }

  // Connecting state
  if (isRunning && (webrtcState === 'connecting' || webrtcState === 'disconnected') && webrtcAvailable) {
    return (
      <div className="flex flex-col items-center gap-3">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
        <p className="text-sm text-nomados-text-muted">Connecting to workspace...</p>
      </div>
    );
  }

  // Paused state
  if (workspaceState === 'Paused') {
    return (
      <div className="flex flex-col items-center gap-4 max-w-md text-center px-4">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-orange-500/10 border border-orange-500/20">
          <svg className="h-8 w-8 text-orange-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M10 9v6m4-6v6m7-3a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-nomados-text">Workspace Paused</h3>
        <p className="text-sm text-nomados-text-muted">
          Resume this workspace to continue your session.
        </p>
      </div>
    );
  }

  // Stopped state
  if (workspaceState === 'Stopped') {
    return (
      <div className="flex flex-col items-center gap-4 max-w-md text-center px-4">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-red-500/10 border border-red-500/20">
          <svg className="h-8 w-8 text-red-400" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
            <path strokeLinecap="round" strokeLinejoin="round" d="M9 10a1 1 0 011-1h4a1 1 0 011 1v4a1 1 0 01-1 1h-4a1 1 0 01-1-1v-4z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-nomados-text">Workspace Stopped</h3>
        <p className="text-sm text-nomados-text-muted">
          Start this workspace to begin a new session.
        </p>
      </div>
    );
  }

  // Running — show stream viewer
  const useNoVNC = !webrtcAvailable || webrtcState === 'failed';
  const noVNCUrl = novncPort
    ? `http://localhost:${novncPort}/vnc.html?autoconnect=true&path=websockify&resize=scale`
    : null;

  // noVNC fallback via iframe
  if (useNoVNC && noVNCUrl) {
    return (
      <div className="relative w-full h-full flex flex-col">
        <div className="absolute top-2 right-2 z-10">
          <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium bg-blue-500/20 text-blue-400">
            noVNC
          </span>
        </div>
        <iframe
          src={noVNCUrl}
          className="flex-1 w-full h-full border-0"
          title="Workspace Stream (noVNC)"
        />
      </div>
    );
  }

  if (useNoVNC && !noVNCUrl) {
    // noVNC not available, show placeholder
    return (
      <div className="relative w-full h-full flex items-center justify-center">
        <div className="absolute top-2 right-2 z-10">
          <span className="inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium bg-gray-500/20 text-gray-400">
            Disconnected
          </span>
        </div>
        <div className="flex flex-col items-center gap-4 max-w-md text-center px-4">
          <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-nomados-surface border border-nomados-border">
            <svg className="h-8 w-8 text-nomados-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
              <path strokeLinecap="round" strokeLinejoin="round" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
            </svg>
          </div>
          <h3 className="text-lg font-semibold text-nomados-text">Stream Viewer</h3>
          <p className="text-sm text-nomados-text-muted leading-relaxed">
            The browser stream will connect via WebRTC when the streaming service is available.
          </p>
          <div className="rounded-lg border border-dashed border-nomados-border px-4 py-3 text-xs text-nomados-text-muted">
            webrtc://workspace/{workspaceId}/stream
          </div>
        </div>
      </div>
    );
  }

  // WebRTC connected or reconnecting
  const statusConfig: Record<ConnectionState, { label: string; color: string }> = {
    connected: { label: 'WebRTC', color: 'bg-green-500/20 text-green-400' },
    connecting: { label: 'Connecting...', color: 'bg-yellow-500/20 text-yellow-400' },
    reconnecting: { label: 'Reconnecting...', color: 'bg-yellow-500/20 text-yellow-400' },
    failed: { label: 'WebRTC Failed', color: 'bg-red-500/20 text-red-400' },
    disconnected: { label: 'Disconnected', color: 'bg-gray-500/20 text-gray-400' },
  };

  const status = statusConfig[webrtcState] || statusConfig.disconnected;

  return (
    <div className="relative w-full h-full flex items-center justify-center">
      <div className="absolute top-2 right-2 z-10">
        <span className={`inline-flex items-center rounded-full px-2 py-0.5 text-xs font-medium ${status.color}`}>
          {status.label}
        </span>
      </div>
      <div className="flex flex-col items-center gap-4 max-w-md text-center px-4">
        <div className="flex h-16 w-16 items-center justify-center rounded-2xl bg-nomados-surface border border-nomados-border">
          <svg className="h-8 w-8 text-nomados-primary" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
            <path strokeLinecap="round" strokeLinejoin="round" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
          </svg>
        </div>
        <h3 className="text-lg font-semibold text-nomados-text">Stream Viewer</h3>
        <p className="text-sm text-nomados-text-muted leading-relaxed">
          {webrtcState === 'connected'
            ? 'Connected via WebRTC. The remote browser stream is active.'
            : webrtcState === 'reconnecting'
              ? 'Reconnecting to workspace...'
              : webrtcState === 'connecting'
                ? 'Establishing WebRTC connection to workspace...'
                : 'The browser stream will connect via WebRTC when the streaming service is available.'}
        </p>
        <div className="rounded-lg border border-dashed border-nomados-border px-4 py-3 text-xs text-nomados-text-muted">
          webrtc://workspace/{workspaceId}/stream
        </div>
      </div>
    </div>
  );
}