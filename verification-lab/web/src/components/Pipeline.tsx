import type { PipelineNode } from "../types";
import { useLabStore } from "../store/labStore";

const nodes: { id: PipelineNode; label: string; hint: string }[] = [
  { id: "api", label: "API", hint: "POST /book" },
  { id: "redis", label: "Redis", hint: "queue + hash" },
  { id: "stream", label: "Stream", hint: "bookings.stream" },
  { id: "worker", label: "Worker", hint: "consumer group" },
  { id: "mysql", label: "MySQL", hint: "durable sold" },
];

export function Pipeline() {
  const snap = useLabStore((s) => s.snap);
  const selected = useLabStore((s) => s.selectedNode);
  const setSelected = useLabStore((s) => s.setSelectedNode);

  return (
    <section className="panel pipeline-panel">
      <div className="panel-head">
        <h2>Pipeline</h2>
        <span className="hint">Click a stage for inspector focus</span>
      </div>
      <div className="pipeline-track">
        {nodes.map((n, i) => (
          <button
            type="button"
            key={n.id}
            className={`pipe-node ${selected === n.id ? "selected" : ""}`}
            onClick={() => setSelected(selected === n.id ? null : n.id)}
          >
            <span className="pipe-label">{n.label}</span>
            <span className="pipe-hint">{n.hint}</span>
            {i < nodes.length - 1 && <span className="pipe-arrow" aria-hidden>→</span>}
          </button>
        ))}
      </div>
      <div className="pipeline-metrics">
        <div>success/s {(snap?.api_rates.success_per_sec ?? 0).toFixed(1)}</div>
        <div>pending {snap?.redis.stream_pending ?? snap?.stream_pending ?? "—"}</div>
        <div>gap {snap?.persistence_lag.success_gap ?? "—"}</div>
      </div>
    </section>
  );
}
