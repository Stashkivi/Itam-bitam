import { useState } from 'react';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { getToken, setToken } from '@/api/client';
import { AppShell } from '@/components/layout/AppShell';
import { AdminPage } from '@/pages/AdminPage';

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry:      1,
      staleTime:  30_000,
    },
  },
});

// ── Simple token gate — keeps auth logic out of AppShell ─────────────────────

function TokenGate({ children }: { children: React.ReactNode }) {
  const [token, setLocalToken] = useState(() => getToken());
  const [input, setInput]      = useState('');

  if (token) return <>{children}</>;

  const submit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!input.trim()) return;
    setToken(input.trim());
    setLocalToken(input.trim());
  };

  return (
    <div className="flex h-screen items-center justify-center bg-surface">
      <form
        onSubmit={submit}
        className="w-96 rounded-xl border border-border bg-card p-8 shadow-2xl"
      >
        <h1 className="text-xl font-bold text-slate-100 mb-1">ITAM Dashboard</h1>
        <p className="text-sm text-muted mb-6">Enter your access token to continue</p>
        <label className="block text-xs text-muted mb-1.5">Access Token</label>
        <input
          type="password"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          placeholder="eyJhbGciOiJIUzI1NiJ9…"
          className="
            w-full rounded-md border border-border bg-surface px-3 py-2
            text-sm text-slate-200 font-mono placeholder:text-slate-600
            focus:outline-none focus:ring-2 focus:ring-blue-500
          "
        />
        <button
          type="submit"
          className="
            mt-4 w-full rounded-md bg-blue-600 px-4 py-2 text-sm font-medium
            text-white hover:bg-blue-500 active:bg-blue-700 transition-colors
          "
        >
          Connect
        </button>
      </form>
    </div>
  );
}

export default function App() {
  const isAdmin = window.location.pathname.startsWith('/admin');

  return (
    <QueryClientProvider client={queryClient}>
      {isAdmin ? (
        <AdminPage />
      ) : (
        <TokenGate>
          <AppShell />
        </TokenGate>
      )}
    </QueryClientProvider>
  );
}
