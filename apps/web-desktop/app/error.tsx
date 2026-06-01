'use client';

export default function GlobalError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4 bg-nomados-background">
      <div className="flex h-20 w-20 items-center justify-center rounded-2xl bg-red-500/10 border border-red-500/20 mb-6">
        <svg
          className="h-10 w-10 text-red-400"
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
      <h2 className="text-2xl font-bold text-nomados-text mb-2">Something went wrong</h2>
      <p className="text-nomados-text-muted text-center max-w-md mb-2">
        An unexpected error occurred. Please try again.
      </p>
      {error?.message && (
        <p className="text-sm text-red-400/80 mb-6 font-mono">
          {error.message}
        </p>
      )}
      <button
        onClick={reset}
        className="rounded-lg bg-nomados-primary px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-nomados-primary/90"
      >
        Try Again
      </button>
    </div>
  );
}