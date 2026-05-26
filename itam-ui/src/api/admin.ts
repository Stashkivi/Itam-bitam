import { apiFetch } from './client';

export interface EnrollmentToken {
  id: string;
  org_id: string;
  label: string;
  used: boolean;
  created_at: string;
}

export interface AlertChannel {
  id: string;
  org_id: string;
  channel_type: 'slack' | 'telegram';
  config: SlackConfig | TelegramConfig;
  enabled: boolean;
  created_at: string;
}

export interface SlackConfig {
  webhook_url: string;
}

export interface TelegramConfig {
  bot_token: string;
  chat_id: string;
}

export interface AdminHost {
  id: string;
  org_id: string;
  hostname: string;
  os_family: string;
  arch: string;
  ip_primary: string | null;
  enrolled_at: string;
  last_seen_at: string | null;
  status: string;
}

function adminHeaders(token: string) {
  return { Authorization: `Bearer ${token}` };
}

export async function createEnrollmentToken(adminToken: string, orgId: string, label: string) {
  return apiFetch<{ token: string; note: string }>('/v1/admin/tokens', {
    method: 'POST',
    headers: adminHeaders(adminToken),
    body: JSON.stringify({ org_id: orgId, label }),
  });
}

export async function listEnrollmentTokens(adminToken: string) {
  return apiFetch<EnrollmentToken[]>('/v1/admin/tokens', {
    headers: adminHeaders(adminToken),
  });
}

export async function listAdminHosts(adminToken: string, orgId?: string) {
  const qs = orgId ? `?org_id=${encodeURIComponent(orgId)}` : '';
  return apiFetch<AdminHost[]>(`/v1/admin/hosts${qs}`, {
    headers: adminHeaders(adminToken),
  });
}

export async function approveBaseline(adminToken: string, hostId: string) {
  return apiFetch<{ status: string }>(`/v1/admin/hosts/${hostId}/baseline`, {
    method: 'PUT',
    headers: adminHeaders(adminToken),
  });
}

export async function listChannels(adminToken: string) {
  return apiFetch<AlertChannel[]>('/v1/admin/channels', {
    headers: adminHeaders(adminToken),
  });
}

export async function createSlackChannel(adminToken: string, orgId: string, webhookUrl: string) {
  return apiFetch<{ id: string }>('/v1/admin/channels', {
    method: 'POST',
    headers: adminHeaders(adminToken),
    body: JSON.stringify({ org_id: orgId, channel_type: 'slack', config: { webhook_url: webhookUrl } }),
  });
}

export async function createTelegramChannel(adminToken: string, orgId: string, botToken: string, chatId: string) {
  return apiFetch<{ id: string }>('/v1/admin/channels', {
    method: 'POST',
    headers: adminHeaders(adminToken),
    body: JSON.stringify({ org_id: orgId, channel_type: 'telegram', config: { bot_token: botToken, chat_id: chatId } }),
  });
}

export async function deleteChannel(adminToken: string, channelId: string) {
  return apiFetch<undefined>(`/v1/admin/channels/${channelId}`, {
    method: 'DELETE',
    headers: adminHeaders(adminToken),
  });
}
