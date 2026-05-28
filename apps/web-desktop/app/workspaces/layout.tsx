'use client';

import { useEffect, useState } from 'react';
import { useRouter } from 'next/navigation';
import { isAuthenticated } from '../../lib/auth';

export default function WorkspacesLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.push('/login');
      return;
    }
    setReady(true);
  }, [router]);

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-nomados-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
      </div>
    );
  }

  return <>{children}</>;
}