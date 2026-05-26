// ── Domain types ──────────────────────────────────────────────────────────────

export interface Host {
  id: string;
  hostname: string;
  os_family: string;
  arch: string;
  ip_primary: string | null;
  last_seen_at: string | null;
  status: 'active' | 'inactive' | 'quarantined';
}

export interface Service {
  name: string;
  status: 'running' | 'stopped' | 'failed' | 'pending' | 'unknown';
  version?: string;
  pid?: number;
}

export interface Process {
  pid: number;
  ppid: number;
  name: string;
  exe: string;
  username: string;
  cpu_pct: number;
  mem_rss_mb: number;
  open_ports?: number[];
  status: string;
}

export interface Port {
  port: number;
  protocol: 'tcp' | 'udp';
  pid: number;
  bind_addr: string;
  process_name?: string;
}

export interface NetIface {
  name: string;
  mac: string;
  addresses: string[];
  is_up: boolean;
}

// Full dependency map returned by GET /v1/hosts/:id/graph
export interface DependencyMap {
  host: Host;
  services: Service[];
  processes: Process[];
  open_ports: Port[];
  interfaces: NetIface[];
}

// ── Anomaly types ─────────────────────────────────────────────────────────────

export type Severity = 'CRITICAL' | 'HIGH' | 'MEDIUM' | 'LOW';

export interface Anomaly {
  id: number;
  host_uuid: string;
  hostname: string;
  rule_id: string;
  severity: Severity;
  entity_type: string;
  entity_key: string;
  details?: Record<string, unknown>;
  detected_at: string;
}

// ── Graph node data shapes ────────────────────────────────────────────────────

export interface HostNodeData {
  host: Host;
  anomalySeverity?: Severity;
}

export interface ServiceNodeData {
  service: Service;
  hostId: string;
  anomalySeverity?: Severity;
}

export interface ProcessNodeData {
  process: Process;
  hostId: string;
  collapsed: boolean;
  anomalySeverity?: Severity;
}

export interface PortNodeData {
  port: Port;
  hostId: string;
  isApproved: boolean;
  anomalySeverity?: Severity;
}

// ── WebSocket message ─────────────────────────────────────────────────────────

export interface WsMessage {
  event: 'anomaly.detected' | 'anomaly.resolved' | 'ping';
  payload?: Anomaly;
}
