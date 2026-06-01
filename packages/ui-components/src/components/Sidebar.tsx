'use client';

import { useState, useEffect, useCallback } from 'react';
import { Button } from './Button';

interface SidebarProps {
  activePath: string;
  onNavigate: (path: string) => void;
  onLogout: () => void;
  userName?: string;
}

const navItems = [
  {
    path: '/workspaces',
    label: 'Workspaces',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z" />
      </svg>
    ),
  },
  {
    path: '/settings/devices',
    label: 'Devices & Security',
    icon: (
      <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={1.5}>
        <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 5.25a3 3 0 013 3m3 0a6 6 0 01-7.029 5.912c-.563-.097-1.159.026-1.639.403l-1.832 1.38a4.5 4.5 0 01-6.364-6.364l1.38-1.832c.377-.48.5-1.076.403-1.639A6 6 0 1121.75 8.25z" />
      </svg>
    ),
  },
];

export function Sidebar({ activePath, onNavigate, onLogout, userName }: SidebarProps) {
  const [expanded, setExpanded] = useState(false);
  const [mobileOpen, setMobileOpen] = useState(false);

  const handleNavigate = useCallback((path: string) => {
    onNavigate(path);
    setMobileOpen(false);
  }, [onNavigate]);

  // Close mobile sidebar on escape
  useEffect(() => {
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setMobileOpen(false);
    };
    window.addEventListener('keydown', handleKey);
    return () => window.removeEventListener('keydown', handleKey);
  }, []);

  const initials = userName ? userName.charAt(0).toUpperCase() : 'U';

  return (
    <>
      {/* Mobile hamburger - only visible below lg */}
      <button
        onClick={() => setMobileOpen(true)}
        className="fixed top-3 left-3 z-40 lg:hidden rounded-md p-2 text-nomados-text hover:bg-nomados-surface"
        aria-label="Open menu"
      >
        <svg className="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
          <path strokeLinecap="round" strokeLinejoin="round" d="M3.75 6.75h16.5M3.75 12h16.5m-16.5 5.25h16.5" />
        </svg>
      </button>

      {/* Mobile overlay */}
      {mobileOpen && (
        <div
          className="fixed inset-0 z-40 bg-black/50 lg:hidden"
          onClick={() => setMobileOpen(false)}
        />
      )}

      {/* Mobile sidebar */}
      <aside
        className={`fixed inset-y-0 left-0 z-50 w-64 transform bg-nomados-surface border-r border-nomados-border transition-transform duration-200 lg:hidden ${
          mobileOpen ? 'translate-x-0' : '-translate-x-full'
        }`}
      >
        <div className="flex h-full flex-col">
          <div className="flex items-center justify-between border-b border-nomados-border px-4 py-3">
            <span className="text-lg font-bold text-nomados-text">NomadOS</span>
            <button
              onClick={() => setMobileOpen(false)}
              className="rounded-md p-1 text-nomados-text-muted hover:text-nomados-text"
              aria-label="Close menu"
            >
              <svg className="h-5 w-5" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M6 18L18 6M6 6l12 12" />
              </svg>
            </button>
          </div>
          <nav className="flex-1 space-y-1 px-2 py-4">
            {navItems.map((item) => (
              <button
                key={item.path}
                onClick={() => handleNavigate(item.path)}
                className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                  activePath === item.path
                    ? 'bg-nomados-primary/10 text-nomados-primary'
                    : 'text-nomados-text-muted hover:bg-nomados-primary/5 hover:text-nomados-text'
                }`}
              >
                {item.icon}
                <span>{item.label}</span>
              </button>
            ))}
          </nav>
          <div className="border-t border-nomados-border p-4">
            <div className="flex items-center gap-3">
              <div className="flex h-8 w-8 items-center justify-center rounded-full bg-nomados-primary/10 text-sm font-medium text-nomados-primary">
                {initials}
              </div>
              <div className="flex-1 min-w-0">
                <p className="truncate text-sm text-nomados-text">{userName || 'User'}</p>
              </div>
              <button
                onClick={onLogout}
                className="rounded-md p-1 text-nomados-text-muted hover:text-nomados-text"
                aria-label="Sign out"
              >
                <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                  <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" />
                </svg>
              </button>
            </div>
          </div>
        </div>
      </aside>

      {/* Desktop sidebar */}
      <aside
        onMouseEnter={() => setExpanded(true)}
        onMouseLeave={() => setExpanded(false)}
        className={`hidden lg:flex flex-col h-screen border-r border-nomados-border bg-nomados-surface transition-all duration-200 ${
          expanded ? 'w-60' : 'w-16'
        }`}
      >
        <div className="flex items-center gap-2 border-b border-nomados-border px-4 py-3 h-14">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-nomados-primary/10">
            <span className="text-sm font-bold text-nomados-primary">N</span>
          </div>
          <span className={`overflow-hidden whitespace-nowrap transition-opacity duration-200 ${
            expanded ? 'opacity-100' : 'opacity-0'
          } text-lg font-bold text-nomados-text`}>
            NomadOS
          </span>
        </div>
        <nav className="flex-1 space-y-1 px-2 py-4">
          {navItems.map((item) => (
            <button
              key={item.path}
              onClick={() => onNavigate(item.path)}
              title={item.label}
              className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors ${
                activePath === item.path
                  ? 'bg-nomados-primary/10 text-nomados-primary'
                  : 'text-nomados-text-muted hover:bg-nomados-primary/5 hover:text-nomados-text'
              }`}
            >
              {item.icon}
              <span className={`overflow-hidden whitespace-nowrap transition-opacity duration-200 ${
                expanded ? 'opacity-100' : 'opacity-0'
              }`}>
                {item.label}
              </span>
            </button>
          ))}
        </nav>
        <div className="border-t border-nomados-border p-3">
          <div className="flex items-center gap-3">
            <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-full bg-nomados-primary/10 text-sm font-medium text-nomados-primary">
              {initials}
            </div>
            <button
              onClick={onLogout}
              title="Sign out"
              className={`overflow-hidden rounded-md p-1.5 text-nomados-text-muted hover:text-nomados-text transition-colors ${
                expanded ? 'opacity-100' : 'opacity-0'
              }`}
              aria-label="Sign out"
            >
              <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor" strokeWidth={2}>
                <path strokeLinecap="round" strokeLinejoin="round" d="M15.75 9V5.25A2.25 2.25 0 0013.5 3h-6a2.25 2.25 0 00-2.25 2.25v13.5A2.25 2.25 0 007.5 21h6a2.25 2.25 0 002.25-2.25V15m3 0l3-3m0 0l-3-3m3 3H9" />
              </svg>
            </button>
          </div>
        </div>
      </aside>
    </>
  );
}