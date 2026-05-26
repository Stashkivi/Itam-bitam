import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { createEnrollmentToken, listEnrollmentTokens } from '@/api/admin';

interface Props { adminToken: string }

export function TokenSection({ adminToken }: Props) {
  const qc = useQueryClient();
  const [orgId, setOrgId] = useState('');
  const [label, setLabel] = useState('');
  const [newToken, setNewToken] = useState<string | null>(null);

  const { data: tokens, isLoading } = useQuery({
    queryKey: ['admin', 'tokens', adminToken],
    queryFn: () => listEnrollmentTokens(adminToken),
  });

  const createMutation = useMutation({
    mutationFn: () => createEnrollmentToken(adminToken, orgId, label),
    onSuccess: (res) => {
      setNewToken(res?.token ?? null);
      qc.invalidateQueries({ queryKey: ['admin', 'tokens'] });
      setOrgId('');
      setLabel('');
    },
  });

  return (
    <div className="space-y-6">
      <section className="rounded-xl border border-border bg-card p-5">
        <h2 className="mb-4 text-sm font-semibold text-slate-300">Generate Enrollment Token</h2>
        <div className="flex gap-3">
          <input
            placeholder="Org ID"
            value={orgId}
            onChange={(e) => setOrgId(e.target.value)}
            className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
          />
          <input
            placeholder="Label (e.g. prod-server-01)"
            value={label}
            onChange={(e) => setLabel(e.target.value)}
            className="flex-1 rounded-lg border border-border bg-background px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
          />
          <button
            onClick={() => createMutation.mutate()}
            disabled={!orgId || createMutation.isPending}
            className="rounded-lg bg-blue-600 px-4 py-2 text-sm font-medium text-white hover:bg-blue-500 disabled:opacity-40"
          >
            {createMutation.isPending ? 'Creating…' : 'Create'}
          </button>
        </div>

        {newToken && (
          <div className="mt-4 rounded-lg border border-emerald-700 bg-emerald-950 p-3">
            <p className="mb-1 text-xs text-emerald-400 font-medium">Token (shown once — copy now)</p>
            <code className="break-all text-xs text-emerald-200">{newToken}</code>
            <button
              onClick={() => navigator.clipboard.writeText(newToken)}
              className="mt-2 block text-xs text-emerald-400 hover:text-emerald-300"
            >
              Copy to clipboard
            </button>
          </div>
        )}
      </section>

      <section className="rounded-xl border border-border bg-card p-5">
        <h2 className="mb-4 text-sm font-semibold text-slate-300">Existing Tokens</h2>
        {isLoading ? (
          <p className="text-xs text-muted">Loading…</p>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-muted border-b border-border">
                <th className="pb-2 pr-4">Label</th>
                <th className="pb-2 pr-4">Org</th>
                <th className="pb-2 pr-4">Created</th>
                <th className="pb-2">Used</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {tokens?.map((t) => (
                <tr key={t.id} className="text-slate-300">
                  <td className="py-2 pr-4 font-mono">{t.label || '—'}</td>
                  <td className="py-2 pr-4">{t.org_id}</td>
                  <td className="py-2 pr-4">{new Date(t.created_at).toLocaleDateString()}</td>
                  <td className="py-2">
                    <span className={t.used ? 'text-slate-500' : 'text-emerald-400'}>
                      {t.used ? 'used' : 'available'}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </section>
    </div>
  );
}
