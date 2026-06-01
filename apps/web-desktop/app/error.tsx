'use client';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex min-h-screen items-center justify-center bg-nomados-background">
      <div className="text-center">
        <div className="flex h-12 w-12 mx-auto items-center justify-center rounded-full bg-red-500/10 mb-4">
          <svg
            className="h-6 w-6 text-red-400"
            fill="none"
            viewBox="0 0 24 24"
            stroke="currentColor"
            strokeWidth={1.5}
          >
            <path
              strokeLinecap="round"
              strokeLinejoin="round"
              d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L12.46 4.128c-.87-1.503-3.025-1.503-3.895 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"
            />
          </svg>
        </div>
        <h2 className="text-lg font-semibold text-nomados-text mb-2">Something went wrong</h2>
        <p className="text-sm text-nomados-text-muted mb-4">
          {error.message || 'An unexpected error occurred.'}
        </p>
        <button
          onClick={reset}
          className="rounded-lg bg-nomados-primary px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-nomados-primary/90"
        >
          Try Again
        </button>
      </div>
    </div>
  );
}