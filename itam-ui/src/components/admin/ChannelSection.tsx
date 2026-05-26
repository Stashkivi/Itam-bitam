import { useState } from 'react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { listChannels, createSlackChannel, createTelegramChannel, deleteChannel } from '@/api/admin';

interface Props { adminToken: string }

export function ChannelSection({ adminToken }: Props) {
  const qc = useQueryClient();
  const [type, setType] = useState<'slack' | 'telegram'>('slack');
  const [orgId, setOrgId] = useState('');
  const [webhookUrl, setWebhookUrl] = useState('');
  const [botToken, setBotToken] = useState('');
  const [chatId, setChatId] = useState('');

  const { data: channels, isLoading } = useQuery({
    queryKey: ['admin', 'channels', adminToken],
    queryFn: () => listChannels(adminToken),
  });

  const createMutation = useMutation({
    mutationFn: () =>
      type === 'slack'
        ? createSlackChannel(adminToken, orgId, webhookUrl)
        : createTelegramChannel(adminToken, orgId, botToken, chatId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['admin', 'channels'] });
      setOrgId(''); setWebhookUrl(''); setBotToken(''); setChatId('');
    },
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteChannel(adminToken, id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['admin', 'channels'] }),
  });

  return (
    <div className="space-y-6">
      <section className="rounded-xl border border-border bg-card p-5">
        <h2 className="mb-4 text-sm font-semibold text-slate-300">Add Alert Channel</h2>

        <div className="mb-4 flex gap-2">
          {(['slack', 'telegram'] as const).map((t) => (
            <button
              key={t}
              onClick={() => setType(t)}
              className={`rounded-md px-3 py-1 text-xs capitalize transition-colors ${
                type === t ? 'bg-indigo-600 text-white' : 'text-muted hover:text-slate-300'
              }`}
            >
              {t}
            </button>
          ))}
        </div>

        <div className="space-y-3">
          <input
            placeholder="Org ID"
            value={orgId}
            onChange={(e) => setOrgId(e.target.value)}
            className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
          />
          {type === 'slack' ? (
            <input
              placeholder="Slack Webhook URL"
              value={webhookUrl}
              onChange={(e) => setWebhookUrl(e.target.value)}
              className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
            />
          ) : (
            <>
              <input
                placeholder="Bot Token"
                value={botToken}
                onChange={(e) => setBotToken(e.target.value)}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
              />
              <input
                placeholder="Chat ID"
                value={chatId}
                onChange={(e) => setChatId(e.target.value)}
                className="w-full rounded-lg border border-border bg-background px-3 py-2 text-sm text-slate-100 outline-none focus:border-blue-500"
              />
            </>
          )}
          <button
            onClick={() => createMutation.mutate()}
            disabled={!orgId || createMutation.isPending}
            className="rounded-lg bg-indigo-600 px-4 py-2 text-sm font-medium text-white hover:bg-indigo-500 disabled:opacity-40"
          >
            {createMutation.isPending ? 'Adding…' : 'Add Channel'}
          </button>
        </div>
      </section>

      <section className="rounded-xl border border-border bg-card p-5">
        <h2 className="mb-4 text-sm font-semibold text-slate-300">Active Channels</h2>
        {isLoading ? (
          <p className="text-xs text-muted">Loading…</p>
        ) : (
          <div className="space-y-2">
            {channels?.map((ch) => (
              <div
                key={ch.id}
                className="flex items-center justify-between rounded-lg border border-border px-4 py-3"
              >
                <div>
                  <span className={`mr-2 inline-block rounded px-1.5 py-0.5 text-xs font-medium ${
                    ch.channel_type === 'slack' ? 'bg-purple-900 text-purple-300' : 'bg-sky-900 text-sky-300'
                  }`}>
                    {ch.channel_type}
                  </span>
                  <span className="text-sm text-slate-300">{ch.org_id}</span>
                  {!ch.enabled && <span className="ml-2 text-xs text-amber-400">disabled</span>}
                </div>
                <button
                  onClick={() => deleteMutation.mutate(ch.id)}
                  className="text-xs text-red-400 hover:text-red-300"
                >
                  Remove
                </button>
              </div>
            ))}
          </div>
        )}
      </section>
    </div>
  );
}
