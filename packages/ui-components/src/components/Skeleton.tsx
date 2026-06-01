'use client';

export function Skeleton({ className = '' }: { className?: string }) {
  return (
    <div
      className={`animate-pulse rounded bg-nomados-border/30 ${className}`}
    />
  );
}

export function WorkspaceCardSkeleton() {
  return (
    <div className="rounded-lg border border-nomados-border bg-nomados-surface p-6">
      <div className="flex items-start justify-between mb-3">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-5 w-16 rounded-full" />
      </div>
      <Skeleton className="h-4 w-24 mb-4" />
      <div className="flex gap-2 pt-3 border-t border-nomados-border">
        <Skeleton className="h-8 w-20 rounded-lg" />
        <Skeleton className="h-8 w-16 rounded-lg" />
        <div className="flex-1" />
        <Skeleton className="h-8 w-16 rounded-lg" />
      </div>
    </div>
  );
}

export function WorkspaceDetailSkeleton() {
  return (
    <div className="flex h-screen flex-col bg-nomados-background">
      {/* Header skeleton */}
      <div className="flex items-center justify-between border-b border-nomados-border bg-nomados-surface px-4 py-3">
        <div className="flex items-center gap-3">
          <Skeleton className="h-8 w-16 rounded-lg" />
          <Skeleton className="h-5 w-40" />
          <Skeleton className="h-5 w-16 rounded-full" />
        </div>
        <div className="flex items-center gap-2">
          <Skeleton className="h-8 w-16 rounded-lg" />
          <Skeleton className="h-8 w-16 rounded-lg" />
        </div>
      </div>
      {/* Main content skeleton */}
      <div className="flex flex-1 overflow-hidden">
        <div className="flex-1 flex items-center justify-center bg-black">
          <Skeleton className="h-64 w-3/4 rounded-lg" />
        </div>
        {/* Sidebar skeleton */}
        <div className="w-72 border-l border-nomados-border bg-nomados-surface p-4">
          <div className="space-y-4">
            <div>
              <Skeleton className="h-3 w-16 mb-2" />
              <Skeleton className="h-4 w-48" />
            </div>
            <div>
              <Skeleton className="h-3 w-16 mb-2" />
              <Skeleton className="h-5 w-16 rounded-full" />
            </div>
            <div>
              <Skeleton className="h-3 w-16 mb-2" />
              <Skeleton className="h-4 w-32" />
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export function FileListSkeleton() {
  return (
    <div className="flex flex-col gap-1">
      {Array.from({ length: 3 }).map((_, i) => (
        <div key={i} className="flex items-center gap-2 rounded-lg px-3 py-2">
          <Skeleton className="h-4 w-4 shrink-0 rounded" />
          <div className="flex-1 min-w-0">
            <Skeleton className="h-4 w-40 mb-1" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
      ))}
    </div>
  );
}