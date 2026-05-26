/**
 * Mock data for Phase 1 development.
 * These types and sample data will be replaced by real API calls when the backend is ready.
 */

export interface MockUser {
  id: string;
  username: string;
  status: 'active' | 'suspended' | 'pending';
  deviceCount: number;
  lastLogin: string;
  createdAt: string;
}

export interface MockSession {
  id: string;
  userId: string;
  username: string;
  device: string;
  ipAddress: string;
  startedAt: string;
  lastActivity: string;
}

export interface MockAuditEvent {
  id: string;
  timestamp: string;
  userId: string;
  username: string;
  action: 'login' | 'logout' | 'mfa_challenge' | 'recovery_initiated' | 'passkey_registered' | 'passkey_removed' | 'session_revoked';
  details: string;
  ipAddress: string;
}

export const mockUsers: MockUser[] = [
  {
    id: 'usr_01',
    username: 'alice',
    status: 'active',
    deviceCount: 2,
    lastLogin: '2025-05-25T14:30:00Z',
    createdAt: '2025-01-15T09:00:00Z',
  },
  {
    id: 'usr_02',
    username: 'bob',
    status: 'active',
    deviceCount: 1,
    lastLogin: '2025-05-24T18:45:00Z',
    createdAt: '2025-02-20T11:30:00Z',
  },
  {
    id: 'usr_03',
    username: 'charlie',
    status: 'suspended',
    deviceCount: 0,
    lastLogin: '2025-04-10T08:15:00Z',
    createdAt: '2025-03-05T16:00:00Z',
  },
  {
    id: 'usr_04',
    username: 'diana',
    status: 'active',
    deviceCount: 3,
    lastLogin: '2025-05-25T10:00:00Z',
    createdAt: '2025-01-28T13:45:00Z',
  },
  {
    id: 'usr_05',
    username: 'eve',
    status: 'pending',
    deviceCount: 0,
    lastLogin: 'Never',
    createdAt: '2025-05-24T20:00:00Z',
  },
];

export const mockSessions: MockSession[] = [
  {
    id: 'ses_01',
    userId: 'usr_01',
    username: 'alice',
    device: 'MacBook Pro — Chrome',
    ipAddress: '192.168.1.42',
    startedAt: '2025-05-25T14:30:00Z',
    lastActivity: '2025-05-25T14:55:00Z',
  },
  {
    id: 'ses_02',
    userId: 'usr_01',
    username: 'alice',
    device: 'iPhone 15 — Safari',
    ipAddress: '10.0.0.23',
    startedAt: '2025-05-25T12:00:00Z',
    lastActivity: '2025-05-25T13:10:00Z',
  },
  {
    id: 'ses_03',
    userId: 'usr_04',
    username: 'diana',
    device: 'ThinkPad X1 — Firefox',
    ipAddress: '172.16.0.8',
    startedAt: '2025-05-25T10:00:00Z',
    lastActivity: '2025-05-25T14:20:00Z',
  },
];

export const mockAuditEvents: MockAuditEvent[] = [
  {
    id: 'evt_01',
    timestamp: '2025-05-25T14:55:00Z',
    userId: 'usr_01',
    username: 'alice',
    action: 'login',
    details: 'Passkey authentication succeeded',
    ipAddress: '192.168.1.42',
  },
  {
    id: 'evt_02',
    timestamp: '2025-05-25T14:30:00Z',
    userId: 'usr_01',
    username: 'alice',
    action: 'mfa_challenge',
    details: 'WebAuthn challenge issued',
    ipAddress: '192.168.1.42',
  },
  {
    id: 'evt_03',
    timestamp: '2025-05-25T13:10:00Z',
    userId: 'usr_01',
    username: 'alice',
    action: 'login',
    details: 'Passkey authentication succeeded (mobile)',
    ipAddress: '10.0.0.23',
  },
  {
    id: 'evt_04',
    timestamp: '2025-05-25T14:20:00Z',
    userId: 'usr_04',
    username: 'diana',
    action: 'passkey_registered',
    details: 'New passkey registered: YubiKey 5',
    ipAddress: '172.16.0.8',
  },
  {
    id: 'evt_05',
    timestamp: '2025-05-25T10:00:00Z',
    userId: 'usr_04',
    username: 'diana',
    action: 'login',
    details: 'Passkey authentication succeeded',
    ipAddress: '172.16.0.8',
  },
  {
    id: 'evt_06',
    timestamp: '2025-05-24T20:00:00Z',
    userId: 'usr_05',
    username: 'eve',
    action: 'recovery_initiated',
    details: 'Account recovery started — recovery key provided',
    ipAddress: '203.0.113.15',
  },
  {
    id: 'evt_07',
    timestamp: '2025-05-25T12:00:00Z',
    userId: 'usr_01',
    username: 'alice',
    action: 'logout',
    details: 'Session ended by user',
    ipAddress: '10.0.0.23',
  },
  {
    id: 'evt_08',
    timestamp: '2025-04-10T08:15:00Z',
    userId: 'usr_03',
    username: 'charlie',
    action: 'session_revoked',
    details: 'All sessions revoked by admin — account suspended',
    ipAddress: '10.0.0.1',
  },
];