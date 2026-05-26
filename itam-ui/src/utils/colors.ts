import type { Severity } from '@/types';

// ── Severity palette ──────────────────────────────────────────────────────────

export const SEVERITY_BORDER: Record<Severity, string> = {
  CRITICAL: 'border-red-500',
  HIGH:     'border-orange-500',
  MEDIUM:   'border-yellow-400',
  LOW:      'border-blue-400',
};

export const SEVERITY_TEXT: Record<Severity, string> = {
  CRITICAL: 'text-red-400',
  HIGH:     'text-orange-400',
  MEDIUM:   'text-yellow-400',
  LOW:      'text-blue-400',
};

export const SEVERITY_BG: Record<Severity, string> = {
  CRITICAL: 'bg-red-500/20',
  HIGH:     'bg-orange-500/20',
  MEDIUM:   'bg-yellow-400/20',
  LOW:      'bg-blue-400/20',
};

// CSS variable for pulse-ring animation (set on the node wrapper)
export const SEVERITY_RING_COLOR: Record<Severity, string> = {
  CRITICAL: 'rgba(239,68,68,0.8)',
  HIGH:     'rgba(249,115,22,0.8)',
  MEDIUM:   'rgba(250,204,21,0.8)',
  LOW:      'rgba(96,165,250,0.8)',
};

// ── Service status ────────────────────────────────────────────────────────────

export const STATUS_DOT: Record<string, string> = {
  running:  'bg-emerald-400',
  stopped:  'bg-slate-500',
  failed:   'bg-red-500',
  pending:  'bg-yellow-400',
  unknown:  'bg-slate-600',
};

// ── Well-known service abbreviations for node icons ───────────────────────────

const SERVICE_ABBREV: Record<string, string> = {
  nginx:         'NG',
  apache2:       'AP',
  httpd:         'AP',
  postgresql:    'PG',
  postgres:      'PG',
  mysql:         'MY',
  mariadb:       'MB',
  redis:         'RD',
  mongodb:       'MG',
  docker:        'DK',
  containerd:    'CT',
  elasticsearch: 'ES',
  kibana:        'KB',
  grafana:       'GF',
  prometheus:    'PR',
  sshd:          'SH',
  cron:          'CR',
  systemd:       'SD',
};

const SERVICE_COLOR: Record<string, string> = {
  nginx:         'bg-emerald-700',
  apache2:       'bg-red-800',
  httpd:         'bg-red-800',
  postgresql:    'bg-blue-800',
  postgres:      'bg-blue-800',
  mysql:         'bg-orange-800',
  mariadb:       'bg-orange-800',
  redis:         'bg-red-700',
  mongodb:       'bg-green-800',
  docker:        'bg-sky-800',
  containerd:    'bg-sky-900',
  elasticsearch: 'bg-yellow-800',
};

export function serviceAbbrev(name: string): string {
  const lower = name.toLowerCase();
  for (const [key, abbrev] of Object.entries(SERVICE_ABBREV)) {
    if (lower.includes(key)) return abbrev;
  }
  return name.slice(0, 2).toUpperCase();
}

export function serviceColor(name: string): string {
  const lower = name.toLowerCase();
  for (const [key, color] of Object.entries(SERVICE_COLOR)) {
    if (lower.includes(key)) return color;
  }
  return 'bg-slate-700';
}

// ── Port helpers ──────────────────────────────────────────────────────────────

export function portLabel(port: number): string {
  const WELL_KNOWN: Record<number, string> = {
    22:   'SSH',
    80:   'HTTP',
    443:  'HTTPS',
    3306: 'MySQL',
    5432: 'PG',
    6379: 'Redis',
    8080: 'HTTP-alt',
    9200: 'ES',
    27017:'Mongo',
  };
  return WELL_KNOWN[port] ?? String(port);
}

export function isExposedBind(addr: string): boolean {
  return addr === '0.0.0.0' || addr === '::' || addr === '*';
}
