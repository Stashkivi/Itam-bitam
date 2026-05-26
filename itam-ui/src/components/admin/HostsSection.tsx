import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listAdminHosts, approveBaseline } from '@/api/admin';

interface Props { adminToken: string }

export function HostsSection({ adminToken }: Props) {
  const qc = useQueryClient();
  const [orgFilter, setOrgFilter] = useState('');
  const [approving, setApproving] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  const { data: hosts, isLoading } = useQuery({
    queryKey: ['admin', 'hosts', adminToken, orgFilter],
    queryFn: () => listAdminHosts(adminToken, orgFilter || undefined),
  });

  const baselineMutation = useMutation({
    mutationFn: (hostId: string) => approveBaseline(adminToken, hostId),
    onSuccess: (_, hostId) => {
      setMessage(`Baseline approved for ${hostId}`);
      setApproving(null);
      qc.invalidateQueries({ queryKey: ['admin', 'hosts'] });
      setTimeout(() => setMessage(null), 4000);
    },
    onError: (e) => {
      setMessage(`Error: ${(e as Error).message}`);
      setApproving(null);
    },
  });

  return (
    <div className="space-y-4">
      <div className="flex items-center gap-3">
        <input
          placeholder="Filter by org ID"
          value={orgFilter}
          onChange={(e) => setOrgFilter(e.target.value)}
          className="w-64 rounded-lg border border-border bg-card px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
        />
        {message && (
          <span className="text-xs text-emerald-400">{message}</span>
        )}
      </div>

      <section className="rounded-xl border border-border bg-card p-5">
        {isLoading ? (
          <p className="text-xs text-muted">Loading…</p>
        ) : (
          <table className="w-full text-xs">
            <thead>
              <tr className="text-left text-muted border-b border-border">
                <th className="pb-2 pr-4">Hostname</th>
                <th className="pb-2 pr-4">Org</th>
                <th className="pb-2 pr-4">OS</th>
                <th className="pb-2 pr-4">IP</th>
                <th className="pb-2 pr-4">Last Seen</th>
                <th className="pb-2">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-border">
              {hosts?.map((h) => (
                <tr key={h.id} className="text-slate-300">
                  <td className="py-2 pr-4 font-mono">{h.hostname}</td>
                  <td className="py-2 pr-4">{h.org_id}</td>
                  <td className="py-2 pr-4">{h.os_family}/{h.arch}</td>
                  <td className="py-2 pr-4">{h.ip_primary ?? '—'}</td>
                  <td className="py-2 pr-4">
                    {h.last_seen_at
                      ? new Date(h.last_seen_at).toLocaleString()
                      : 'never'}
                  </td>
                  <td className="py-2">
                    <button
                      onClick={() => {
                        setApproving(h.id);
                        baselineMutation.mutate(h.id);
                      }}
                      disabled={approving === h.id}
                      className="rounded bg-emerald-700 px-2 py-1 text-xs text-white hover:bg-emerald-600 disabled:opacity-40"
                    >
                      {approving === h.id ? 'Approving…' : 'Approve baseline'}
                    </button>
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
