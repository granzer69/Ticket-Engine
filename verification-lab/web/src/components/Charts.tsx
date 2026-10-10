import { useMemo } from "react";
import { useLabStore } from "../store/labStore";
import type { ChartPoint } from "../store/labStore";

function downsample<T>(data: T[], max: number): T[] {
  if (data.length <= max) return data;
  const step = data.length / max;
  const out: T[] = [];
  for (let i = 0; i < max; i++) {
    out.push(data[Math.floor(i * step)]);
  }
  return out;
}

function lastVal(values: number[]): number {
  return values.length ? values[values.length - 1] : 0;
}

function Sparkline({ values, color }: { values: number[]; color: string }) {
  const w = 280;
  const h = 48;
  const { pts, flat } = useMemo(() => {
    if (!values.length) return { pts: "", flat: true };
    const min = Math.min(...values);
    const max = Math.max(...values);
    const span = max - min;
    const flatLine = span < 1e-9;
    const top = flatLine ? Math.max(max, 1) : max;
    const bottom = flatLine ? 0 : min;
    const range = top - bottom || 1;
    if (values.length === 1) {
      const y = h / 2;
      return { pts: `0,${y} ${w},${y}`, flat: flatLine };
    }
    const line = values
      .map((v, i) => {
        const x = (i / (values.length - 1)) * w;
        const norm = flatLine ? 0.5 : (v - bottom) / range;
        const y = h - norm * (h - 8) - 4;
        return `${x},${y}`;
      })
      .join(" ");
    return { pts: line, flat: flatLine };
  }, [values]);

  if (!values.length) {
    return <div className="chart-empty">No samples yet — waiting for SSE</div>;
  }

  return (
    <svg className="spark" viewBox={`0 0 ${w} ${h}`} role="img" aria-label="sparkline">
      {flat && <line x1={0} y1={h / 2} x2={w} y2={h / 2} stroke="#334155" strokeWidth="1" strokeDasharray="4 3" />}
      <polyline fill="none" stroke={color} strokeWidth="1.5" points={pts} />
    </svg>
  );
}

function ChartCard({
  title,
  values,
  color,
  unit,
  format,
}: {
  title: string;
  values: number[];
  color: string;
  unit: string;
  format?: (v: number) => string;
}) {
  const fmt = format ?? ((v: number) => (v >= 1000 ? `${(v / 1000).toFixed(1)}k` : v.toFixed(1)));
  const current = lastVal(values);
  return (
    <div className="chart-card">
      <div className="chart-title">{title}</div>
      <div className="chart-now">
        <span className="chart-now-val">{fmt(current)}</span>
        <span className="chart-now-unit">{unit}</span>
        <span className="chart-now-n">{values.length} pts</span>
      </div>
      <Sparkline values={values} color={color} />
    </div>
  );
}

export function Charts() {
  const chart = useLabStore((s) => s.chart);
  const connected = useLabStore((s) => s.connected);
  const ds = downsample(chart, 120);

  const series = useMemo(() => {
    const pick = (fn: (p: ChartPoint) => number) => ds.map(fn);
    return {
      req: pick((p) => p.reqPerSec),
      success: pick((p) => p.successPerSec),
      fail: pick((p) => p.failPerSec),
      queue: pick((p) => p.queue),
      pending: pick((p) => p.pending),
      sold: pick((p) => p.sold),
    };
  }, [ds]);

  return (
    <section className="panel charts-panel">
      <div className="panel-head">
        <h2>Charts</h2>
        <span className="hint">
          {connected ? "SSE · " : "offline · "}
          {chart.length} samples · display downsampled to 120
        </span>
      </div>
      {chart.length < 2 && (
        <p className="chart-hint muted">
          Charts fill from the telemetry stream during a run. If you started the page after a test finished, start a smoke
          run or keep this tab open during the next heavy load.
        </p>
      )}
      <div className="chart-grid">
        <ChartCard title="API requests / sec (DERIVED)" values={series.req} color="#a78bfa" unit="/s" />
        <ChartCard title="Booking success / sec (DERIVED)" values={series.success} color="#5eead4" unit="/s" />
        <ChartCard title="Booking failures / sec (DERIVED)" values={series.fail} color="#f87171" unit="/s" />
        <ChartCard title="Queue depth (LIVE)" values={series.queue} color="#93c5fd" unit="tickets" format={(v) => String(Math.round(v))} />
        <ChartCard title="Stream pending (LIVE)" values={series.pending} color="#fbbf24" unit="msgs" format={(v) => String(Math.round(v))} />
        <ChartCard title="MySQL sold rows (LIVE)" values={series.sold} color="#4ade80" unit="rows" format={(v) => String(Math.round(v))} />
      </div>
    </section>
  );
}
