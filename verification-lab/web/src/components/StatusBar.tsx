import { useLabStore } from "../store/labStore";
import { ProvenanceBadge } from "./ProvenanceBadge";

export function StatusBar() {
  const snap = useLabStore((s) => s.snap);
  const connected = useLabStore((s) => s.connected);
  const prov = snap?.field_provenance ?? {};

  const lag = snap?.stale_ms ?? 0;
  const lagClass = lag > 800 ? "lag-bad" : lag > 400 ? "lag-warn" : "lag-ok";

  return (
    <header className="status-bar">
      <div className="brand">
        <h1>Verification Lab</h1>
        <span className="sub">Ticket Engine V3 · control plane</span>
      </div>
      <div className="status-pills">
        <span className={`pill ${connected ? "on" : "off"}`}>{connected ? "SSE live" : "SSE reconnecting"}</span>
        <span className={`pill ${lagClass}`}>lag {lag}ms</span>
        <span className="pill muted">seq {snap?.sequence ?? "—"}</span>
        <span className="pill muted">collect {snap?.collect_ms ?? "—"}ms</span>
      </div>
      <div className="target-line">
        Target <code>{snap?.target_base ?? "…"}</code>
        {snap?.active_run_id && (
          <span className="run-tag">run {snap.active_run_id.slice(0, 8)}</span>
        )}
        <ProvenanceBadge label={prov.target_metrics} />
      </div>
    </header>
  );
}
