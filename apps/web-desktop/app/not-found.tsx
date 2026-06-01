import Link from 'next/link';

export default function NotFound() {
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
            d="M12 9v3.75m9-.75a9 9 0 11-18 0 9 9 0 0118 0zm-9 3.75h.008v.008H12v-.008z"
          />
        </svg>
      </div>
      <h1 className="text-3xl font-bold text-nomados-text mb-2">Page Not Found</h1>
      <p className="text-nomados-text-muted text-center max-w-md mb-8">
        The page you&apos;re looking for doesn&apos;t exist or has been moved.
      </p>
      <Link
        href="/workspaces"
        className="rounded-lg bg-nomados-primary px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-nomados-primary/90"
      >
        Go to Workspaces
      </Link>
    </div>
  );
}