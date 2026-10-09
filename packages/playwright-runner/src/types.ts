export interface RunCheckRequest {
  scenario?: string;
  target: string;
  timeoutMs?: number;
  parameters?: Record<string, string>;
}

export type StatusState = "operational" | "degraded" | "outage" | "maintenance";

export interface RunCheckResponse {
  success: boolean;
  state: StatusState;
  latencyMs: number;
  message: string;
  details?: Record<string, unknown>;
}

export interface ScenarioContext {
  target: string;
  timeoutMs: number;
  parameters: Record<string, string>;
}
