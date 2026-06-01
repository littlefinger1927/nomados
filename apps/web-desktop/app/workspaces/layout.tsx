'use client';

import { useEffect, useState } from 'react';
import { useRouter, usePathname } from 'next/navigation';
import { isAuthenticated } from '../../lib/auth';
import { Sidebar } from '@nomados/ui-components';

export default function WorkspacesLayout({ children }: { children: React.ReactNode }) {
  const router = useRouter();
  const pathname = usePathname();
  const [ready, setReady] = useState(false);

  useEffect(() => {
    if (!isAuthenticated()) {
      router.push('/login');
      return;
    }
    setReady(true);
  }, [router]);

  const handleNavigate = (path: string) => {
    router.push(path);
  };

  const handleLogout = () => {
    import('../../lib/auth').then(({ logout }) => logout());
  };

  if (!ready) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-nomados-background">
        <div className="h-8 w-8 animate-spin rounded-full border-2 border-nomados-border border-t-nomados-primary" />
      </div>
    );
  }

  // Determine active path for sidebar highlighting
  const activePath = pathname.startsWith('/settings') ? '/settings/devices' : '/workspaces';

  return (
    <div className="flex h-screen overflow-hidden">
      <Sidebar
        activePath={activePath}
        onNavigate={handleNavigate}
        onLogout={handleLogout}
      />
      <main className="flex-1 overflow-auto">
        {children}
      </main>
    </div>
  );
}