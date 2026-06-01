import Link from 'next/link';

export default function WorkspaceNotFound() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4 bg-nomados-background">
      <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-nomados-surface border border-nomados-border mb-6">
        <svg
          className="h-10 w-10 text-nomados-text-muted"
          fill="none"
          viewBox="0 0 24 24"
          stroke="currentColor"
          strokeWidth={1.5}
        >
          <path
            strokeLinecap="round"
            strokeLinejoin="round"
            d="M20 7l-8-4-8 4m16 0l-8 4m8-4v10l-8 4m0-10L4 7m8 4v10M4 7v10l8 4"
          />
        </svg>
      </div>
      <h1 className="text-3xl font-bold text-nomados-text mb-2">Workspace Not Found</h1>
      <p className="text-nomados-text-muted text-center max-w-md mb-8">
        This workspace doesn&apos;t exist or you don&apos;t have access to it.
      </p>
      <Link
        href="/workspaces"
        className="rounded-lg bg-nomados-primary px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-nomados-primary/90"
      >
        Back to Workspaces
      </Link>
    </div>
  );
}