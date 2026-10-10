export function ProvenanceBadge({ label }: { label?: string }) {
  const v = (label ?? "UNAVAILABLE").toUpperCase();
  const cls = v === "LIVE" ? "live" : v === "DERIVED" ? "derived" : "unavailable";
  return <span className={`badge ${cls}`}>{v}</span>;
}
