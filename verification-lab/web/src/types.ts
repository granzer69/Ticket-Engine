export type TelemetryEvent = {
  seq: number;
  at: string;
  kind: string;
  message: string;
  run_id?: string;
  correlation_id?: string;
  provenance: string;
};

export type Snapshot = {
  at: string;
  sequence: number;
  collect_ms: number;
  stale_ms?: number;
  target_base: string;
  active_run_id?: string;
  correlation_id?: string;
  target_metrics: { total_requests: number; booking_success: number; booking_failures: number };
  api_rates: { requests_per_sec: number; success_per_sec: number; failures_per_sec: number };
  redis: {
    queue_len: number;
    user_hash_len: number;
    stream_len: number;
    stream_pending: number;
    dlq_len: number;
  };
  worker: { consumers: number; pending: number; last_delivered_id?: string };
  mysql: { available: number; sold: number; total: number };
  persistence_lag: { success_gap: number; estimated_ms: number };
  events: TelemetryEvent[];
  field_provenance: Record<string, string>;
  queue_remaining: number;
  stream_pending: number;
};

export type ClaimResult = {
  id: string;
  status: string;
  message: string;
  evidence: Record<string, unknown>;
  registry_tests: string[];
};

export type RunReport = {
  run_id: string;
  preset: string;
  status: string;
  claims?: ClaimResult[];
  http_200?: number;
  http_404?: number;
  errors?: number;
  logical_requests?: number;
  duration_ms?: number;
  latency_p95_ms?: number;
  snapshot_start?: Snapshot;
  snapshot_end?: Snapshot;
  /** Client-side or API error text (POST failure, poll timeout). */
  error?: string;
  extra?: Record<string, unknown>;
};

export type PipelineNode = "api" | "redis" | "stream" | "worker" | "mysql";
