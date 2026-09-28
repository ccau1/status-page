export type StatusState = 'operational' | 'degraded' | 'outage' | 'maintenance' | 'unknown';

export interface FeatureStatus {
  id: string;
  name: string;
  description?: string;
  state: StatusState;
  last_checked: string;
  latency_ms?: number;
  message?: string;
}

export interface CheckResult {
  check_id: string;
  feature: string;
  type: string;
  success: boolean;
  state: StatusState;
  latency_ms: number;
  message?: string;
  timestamp: string;
}

export interface Status {
  tenant: string;
  product: string;
  current_state: StatusState;
  message?: string;
  features: Record<string, FeatureStatus>;
  last_updated: string;
  check_results?: CheckResult[];
}

export interface StatusApiResponse {
  items: Status[];
  count: number;
  tenant?: string;
}

export type IncidentSeverity = 'critical' | 'major' | 'minor' | 'maintenance' | 'info';
export type IncidentState = 'investigating' | 'identified' | 'monitoring' | 'resolved';

export interface IncidentUpdate {
  timestamp: string;
  state: IncidentState;
  message: string;
}

export interface Incident {
  id: string;
  tenant: string;
  product?: string;
  products?: string[];
  check_ids?: string[];
  features?: string[];
  title: string;
  severity: IncidentSeverity;
  state: IncidentState;
  message: string;
  active: boolean;
  created_at: string;
  updated_at: string;
  updates?: IncidentUpdate[];
}

export interface IncidentApiResponse {
  tenant: string;
  incidents: Incident[];
  count: number;
}
