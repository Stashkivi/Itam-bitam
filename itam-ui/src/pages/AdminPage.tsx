import { useState } from 'react';
import { TokenSection } from '@/components/admin/TokenSection';
import { ChannelSection } from '@/components/admin/ChannelSection';
import { HostsSection } from '@/components/admin/HostsSection';

const ADMIN_KEY = 'itam_admin_token';

export function AdminPage() {
  const [adminToken, setAdminToken] = useState(
    () => localStorage.getItem(ADMIN_KEY) ?? '',
  );
  const [input, setInput] = useState('');
  const [tab, setTab] = useState<'tokens' | 'channels' | 'hosts'>('tokens');

  if (!adminToken) {
    return (
      <div className="flex h-screen items-center justify-center bg-background">
        <div className="w-96 rounded-xl border border-border bg-card p-8 shadow-xl">
          <h1 className="mb-6 text-xl font-semibold text-slate-100">Admin Login</h1>
          <input
            type="password"
            placeholder="Admin bearer token"
            value={input}
            onChange={(e) => setInput(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && input) {
                localStorage.setItem(ADMIN_KEY, input);
                setAdminToken(input);
              }
            }}
            className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
          />
          <button
            onClick={() => {
              if (input) {
                localStorage.setItem(ADMIN_KEY, input);
                setAdminToken(input);
              }
            }}
            className="mt-4 w-full rounded-lg bg-blue-600 py-2 text-sm font-medium text-white hover:bg-blue-500"
          >
            Continue
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="flex h-screen flex-col bg-background text-slate-100">
      {/* Header */}
      <header className="flex items-center justify-between border-b border-border px-6 py-3">
        <h1 className="text-lg font-semibold">ITAM Admin</h1>
        <button
          onClick={() => {
            localStorage.removeItem(ADMIN_KEY);
            setAdminToken('');
          }}
          className="text-xs text-muted hover:text-slate-300"
        >
          Sign out
        </button>
      </header>

      {/* Tabs */}
      <nav className="flex gap-1 border-b border-border px-6 py-2">
        {(['tokens', 'channels', 'hosts'] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`rounded-md px-3 py-1.5 text-sm capitalize transition-colors ${
              tab === t
                ? 'bg-blue-600 text-white'
                : 'text-muted hover:text-slate-300'
            }`}
          >
            {t}
          </button>
        ))}
      </nav>

      {/* Content */}
      <main className="flex-1 overflow-auto p-6">
        {tab === 'tokens'   && <TokenSection   adminToken={adminToken} />}
        {tab === 'channels' && <ChannelSection adminToken={adminToken} />}
        {tab === 'hosts'    && <HostsSection   adminToken={adminToken} />}
      </main>
    </div>
  );
}
