import { useLabStore } from "../store/labStore";
import { ProvenanceBadge } from "./ProvenanceBadge";

function Row({ k, v }: { k: string; v: string | number }) {
  return (
    <div className="insp-row">
      <span>{k}</span>
      <span className="val">{v}</span>
    </div>
  );
}

export function Inspectors() {
  const snap = useLabStore((s) => s.snap);
  const focus = useLabStore((s) => s.selectedNode);
  const prov = snap?.field_provenance ?? {};

  return (
    <section className="panel inspectors">
      <div className="panel-head">
        <h2>Inspectors</h2>
        <span className="hint">{focus ? `focused: ${focus}` : "all layers"}</span>
      </div>
      <div className="insp-grid">
        {(!focus || focus === "api") && (
          <div className="insp-card">
            <h3>API <ProvenanceBadge label={prov.target_metrics} /></h3>
            <Row k="requests" v={snap?.target_metrics.total_requests ?? "—"} />
            <Row k="success" v={snap?.target_metrics.booking_success ?? "—"} />
            <Row k="failures" v={snap?.target_metrics.booking_failures ?? "—"} />
            <Row k="req/s" v={(snap?.api_rates.requests_per_sec ?? 0).toFixed(2)} />
          </div>
        )}
        {(!focus || focus === "redis") && (
          <div className="insp-card">
            <h3>Redis <ProvenanceBadge label={prov.redis} /></h3>
            <Row k="queue LLEN" v={snap?.redis.queue_len ?? snap?.queue_remaining ?? "—"} />
            <Row k="hash HLEN" v={snap?.redis.user_hash_len ?? "—"} />
            <Row k="DLQ XLEN" v={snap?.redis.dlq_len ?? "—"} />
          </div>
        )}
        {(!focus || focus === "stream") && (
          <div className="insp-card">
            <h3>Stream <ProvenanceBadge label={prov.stream_pending} /></h3>
            <Row k="XLEN" v={snap?.redis.stream_len ?? "—"} />
            <Row k="XPENDING" v={snap?.redis.stream_pending ?? snap?.stream_pending ?? "—"} />
          </div>
        )}
        {(!focus || focus === "worker") && (
          <div className="insp-card">
            <h3>Worker <ProvenanceBadge label={prov.worker} /></h3>
            <Row k="consumers" v={snap?.worker.consumers ?? "—"} />
            <Row k="pending" v={snap?.worker.pending ?? "—"} />
            <Row k="last ID" v={snap?.worker.last_delivered_id ?? "—"} />
          </div>
        )}
        {(!focus || focus === "mysql") && (
          <div className="insp-card">
            <h3>MySQL <ProvenanceBadge label={prov.mysql} /></h3>
            <Row k="available" v={snap?.mysql.available ?? "—"} />
            <Row k="sold" v={snap?.mysql.sold ?? "—"} />
            <Row k="total" v={snap?.mysql.total ?? "—"} />
            <Row
              k="est lag ms"
              v={
                prov.persistence_lag === "DERIVED"
                  ? (snap?.persistence_lag.estimated_ms ?? 0).toFixed(0)
                  : "—"
              }
            />
          </div>
        )}
      </div>
    </section>
  );
}
