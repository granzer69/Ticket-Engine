import { useMemo } from "react";
import { useLabStore } from "../store/labStore";
import { ProvenanceBadge } from "./ProvenanceBadge";

export function Timeline() {
  const events = useLabStore((s) => s.events);
  const filter = useLabStore((s) => s.eventFilter);
  const setFilter = useLabStore((s) => s.setEventFilter);

  const filtered = useMemo(() => {
    const f = filter.trim().toLowerCase();
    if (!f) return [...events].reverse();
    return events.filter((e) => e.kind.toLowerCase().includes(f) || e.message.toLowerCase().includes(f)).reverse();
  }, [events, filter]);

  return (
    <section className="panel timeline-panel">
      <div className="panel-head">
        <h2>Event timeline</h2>
        <input
          className="filter-input"
          placeholder="Filter kind/message"
          value={filter}
          onChange={(e) => setFilter(e.target.value)}
        />
      </div>
      <ul className="timeline">
        {filtered.length === 0 && <li className="muted">No events yet — run lab scenarios or wait for metric deltas.</li>}
        {filtered.slice(0, 40).map((e) => (
          <li key={e.seq}>
            <span className="ts">{new Date(e.at).toLocaleTimeString()}</span>
            <span className="kind">{e.kind}</span>
            <span className="msg">{e.message}</span>
            <ProvenanceBadge label={e.provenance} />
          </li>
        ))}
      </ul>
    </section>
  );
}
