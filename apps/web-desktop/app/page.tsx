'use client';

import { useEffect } from 'react';

export default function HomePage() {
  useEffect(() => {
    const token = typeof window !== 'undefined'
      ? localStorage.getItem('nomados-session')
      : null;

    if (token) {
      window.location.href = '/workspaces';
    } else {
      window.location.href = '/login';
    }
  }, []);

  return (
    <div className="flex min-h-screen items-center justify-center">
      <div className="flex flex-col items-center gap-3">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
        <p className="text-sm text-nomados-text-muted">Redirecting...</p>
      </div>
    </div>
  );
}