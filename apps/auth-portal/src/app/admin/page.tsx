'use client';

import { useState } from 'react';
import { Button, Card } from '@nomados/ui-components';
import { mockUsers, mockSessions, mockAuditEvents } from '@/lib/mock-data';
import type { MockUser, MockSession, MockAuditEvent } from '@/lib/mock-data';

type AdminTab = 'users' | 'sessions' | 'audit';

function StatusBadge({ status }: { status: MockUser['status'] }) {
  const colors: Record<MockUser['status'], string> = {
    active: 'bg-green-500/20 text-green-400',
    suspended: 'bg-red-500/20 text-red-400',
    pending: 'bg-amber-500/20 text-amber-400',
  };
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${colors[status]}`}>
      {status}
    </span>
  );
}

function ActionBadge({ action }: { action: MockAuditEvent['action'] }) {
  const labels: Record<MockAuditEvent['action'], string> = {
    login: 'Login',
    logout: 'Logout',
    mfa_challenge: 'MFA Challenge',
    recovery_initiated: 'Recovery',
    passkey_registered: 'Passkey Added',
    passkey_removed: 'Passkey Removed',
    session_revoked: 'Session Revoked',
  };
  const colors: Record<MockAuditEvent['action'], string> = {
    login: 'bg-green-500/20 text-green-400',
    logout: 'bg-gray-500/20 text-gray-400',
    mfa_challenge: 'bg-blue-500/20 text-blue-400',
    recovery_initiated: 'bg-amber-500/20 text-amber-400',
    passkey_registered: 'bg-purple-500/20 text-purple-400',
    passkey_removed: 'bg-red-500/20 text-red-400',
    session_revoked: 'bg-red-500/20 text-red-400',
  };
  return (
    <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ${colors[action]}`}>
      {labels[action]}
    </span>
  );
}

function formatDate(dateStr: string): string {
  if (dateStr === 'Never') return 'Never';
  try {
    return new Date(dateStr).toLocaleString();
  } catch {
    return dateStr;
  }
}

export default function AdminPage() {
  const [activeTab, setActiveTab] = useState<AdminTab>('users');
  const [users, setUsers] = useState(mockUsers);
  const [sessions, setSessions] = useState(mockSessions);

  const handleRevokeSession = (sessionId: string) => {
    setSessions((prev) => prev.filter((s) => s.id !== sessionId));
  };

  const handleSuspendUser = (userId: string) => {
    setUsers((prev) =>
      prev.map((u) =>
        u.id === userId ? { ...u, status: 'suspended' as const } : u
      )
    );
  };

  const tabs: { key: AdminTab; label: string }[] = [
    { key: 'users', label: 'User Management' },
    { key: 'sessions', label: 'Active Sessions' },
    { key: 'audit', label: 'Audit Log' },
  ];

  return (
    <div className="min-h-screen bg-nomados-background p-6">
      <div className="mx-auto max-w-6xl">
        {/* Header */}
        <div className="mb-8">
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-lg bg-nomados-primary">
              <svg
                className="h-6 w-6 text-white"
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
            <div>
              <h1 className="text-2xl font-bold text-nomados-text">Auth Admin</h1>
              <p className="text-sm text-nomados-text-muted">NomadOS Authentication Dashboard</p>
            </div>
          </div>
          <div className="mt-3 rounded-lg border border-amber-500/30 bg-amber-500/10 px-4 py-2 text-sm text-amber-400">
            Phase 1 — Showing mock data. Connect to the gateway service for live data.
          </div>
        </div>

        {/* Tabs */}
        <div className="mb-6 flex gap-1 rounded-lg border border-nomados-border bg-nomados-surface p-1">
          {tabs.map((tab) => (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`flex-1 rounded-md px-4 py-2 text-sm font-medium transition-colors ${
                activeTab === tab.key
                  ? 'bg-nomados-primary text-white'
                  : 'text-nomados-text-muted hover:bg-nomados-border hover:text-nomados-text'
              }`}
            >
              {tab.label}
            </button>
          ))}
        </div>

        {/* Users Tab */}
        {activeTab === 'users' && (
          <Card title="User Management" className="border-nomados-border">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-nomados-border">
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Username</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Status</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Devices</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Last Login</th>
                    <th className="pb-3 font-medium text-nomados-text-muted">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {users.map((user) => (
                    <tr key={user.id} className="border-b border-nomados-border/50 last:border-0">
                      <td className="py-3 pr-4 font-medium text-nomados-text">{user.username}</td>
                      <td className="py-3 pr-4">
                        <StatusBadge status={user.status} />
                      </td>
                      <td className="py-3 pr-4 text-nomados-text-muted">{user.deviceCount}</td>
                      <td className="py-3 pr-4 text-nomados-text-muted">{formatDate(user.lastLogin)}</td>
                      <td className="py-3">
                        {user.status !== 'suspended' && (
                          <Button
                            variant="danger"
                            size="sm"
                            onClick={() => handleSuspendUser(user.id)}
                          >
                            Suspend
                          </Button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        )}

        {/* Sessions Tab */}
        {activeTab === 'sessions' && (
          <Card title="Active Sessions" actions={
            <span className="text-sm text-nomados-text-muted">
              {sessions.length} active session{sessions.length !== 1 ? 's' : ''}
            </span>
          } className="border-nomados-border">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-nomados-border">
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">User</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Device</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">IP Address</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Started</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Last Activity</th>
                    <th className="pb-3 font-medium text-nomados-text-muted">Actions</th>
                  </tr>
                </thead>
                <tbody>
                  {sessions.map((session) => (
                    <tr key={session.id} className="border-b border-nomados-border/50 last:border-0">
                      <td className="py-3 pr-4 font-medium text-nomados-text">{session.username}</td>
                      <td className="py-3 pr-4 text-nomados-text-muted">{session.device}</td>
                      <td className="py-3 pr-4 text-nomados-text-muted font-mono text-xs">{session.ipAddress}</td>
                      <td className="py-3 pr-4 text-nomados-text-muted">{formatDate(session.startedAt)}</td>
                      <td className="py-3 pr-4 text-nomados-text-muted">{formatDate(session.lastActivity)}</td>
                      <td className="py-3">
                        <Button
                          variant="danger"
                          size="sm"
                          onClick={() => handleRevokeSession(session.id)}
                        >
                          Revoke
                        </Button>
                      </td>
                    </tr>
                  ))}
                  {sessions.length === 0 && (
                    <tr>
                      <td colSpan={6} className="py-8 text-center text-nomados-text-muted">
                        No active sessions
                      </td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </Card>
        )}

        {/* Audit Tab */}
        {activeTab === 'audit' && (
          <Card title="Audit Log" actions={
            <span className="text-sm text-nomados-text-muted">
              {mockAuditEvents.length} events
            </span>
          } className="border-nomados-border">
            <div className="overflow-x-auto">
              <table className="w-full text-left text-sm">
                <thead>
                  <tr className="border-b border-nomados-border">
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Time</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">User</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Action</th>
                    <th className="pb-3 pr-4 font-medium text-nomados-text-muted">Details</th>
                    <th className="pb-3 font-medium text-nomados-text-muted">IP</th>
                  </tr>
                </thead>
                <tbody>
                  {mockAuditEvents.map((event) => (
                    <tr key={event.id} className="border-b border-nomados-border/50 last:border-0">
                      <td className="py-3 pr-4 text-nomados-text-muted text-xs font-mono">
                        {formatDate(event.timestamp)}
                      </td>
                      <td className="py-3 pr-4 font-medium text-nomados-text">{event.username}</td>
                      <td className="py-3 pr-4">
                        <ActionBadge action={event.action} />
                      </td>
                      <td className="py-3 pr-4 text-nomados-text-muted max-w-xs truncate">{event.details}</td>
                      <td className="py-3 text-nomados-text-muted font-mono text-xs">{event.ipAddress}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </Card>
        )}

        {/* Footer */}
        <div className="mt-6 text-center text-xs text-nomados-text-muted">
          NomadOS Auth Portal — Phase 1 Development Build
        </div>
      </div>
    </div>
  );
}