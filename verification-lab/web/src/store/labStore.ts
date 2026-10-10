import { create } from "zustand";
import type { PipelineNode, RunReport, Snapshot, TelemetryEvent } from "../types";

/** ~15 min of 200ms frames before display downsampling. */
const CHART_CAP = 4500;

export type ChartPoint = {
  t: number;
  seq: number;
  reqPerSec: number;
  successPerSec: number;
  failPerSec: number;
  queue: number;
  pending: number;
  sold: number;
};

type LabState = {
  snap: Snapshot | null;
  connected: boolean;
  lastFrameAt: number;
  chart: ChartPoint[];
  events: TelemetryEvent[];
  eventFilter: string;
  selectedNode: PipelineNode | null;
  run: RunReport | null;
  busy: boolean;
  setSnap: (s: Snapshot) => void;
  setConnected: (v: boolean) => void;
  setEventFilter: (f: string) => void;
  setSelectedNode: (n: PipelineNode | null) => void;
  setRun: (r: RunReport | null) => void;
  setBusy: (b: boolean) => void;
  pushChartFromSnap: (s: Snapshot) => void;
  pushChartFromRunSnapshots: (start: Snapshot, end: Snapshot) => void;
  clearChart: () => void;
  mergeEvents: (ev: TelemetryEvent[]) => void;
};

export const useLabStore = create<LabState>((set, get) => ({
  snap: null,
  connected: false,
  lastFrameAt: 0,
  chart: [],
  events: [],
  eventFilter: "",
  selectedNode: null,
  run: null,
  busy: false,
  setSnap: (s) => set({ snap: s, lastFrameAt: Date.now() }),
  setConnected: (v) => set({ connected: v }),
  setEventFilter: (f) => set({ eventFilter: f }),
  setSelectedNode: (n) => set({ selectedNode: n }),
  setRun: (r) => set({ run: r }),
  setBusy: (b) => set({ busy: b }),
  pushChartFromSnap: (s) => {
    const t = Date.parse(s.at);
    if (!Number.isFinite(t)) return;
    const chart = get().chart;
    if (s.sequence > 0 && chart.length > 0 && chart[chart.length - 1]?.seq === s.sequence) {
      return;
    }
    const pt: ChartPoint = {
      t,
      seq: s.sequence,
      reqPerSec: s.api_rates?.requests_per_sec ?? 0,
      successPerSec: s.api_rates?.success_per_sec ?? 0,
      failPerSec: s.api_rates?.failures_per_sec ?? 0,
      queue: s.redis?.queue_len ?? s.queue_remaining ?? 0,
      pending: s.redis?.stream_pending ?? s.stream_pending ?? 0,
      sold: s.mysql?.sold ?? 0,
    };
    const next = [...chart, pt];
    set({ chart: next.length > CHART_CAP ? next.slice(-CHART_CAP) : next });
  },
  pushChartFromRunSnapshots: (start, end) => {
    const pts: ChartPoint[] = [];
    for (const s of [start, end]) {
      const t = Date.parse(s.at);
      if (!Number.isFinite(t)) continue;
      pts.push({
        t,
        seq: s.sequence,
        reqPerSec: s.api_rates?.requests_per_sec ?? 0,
        successPerSec: s.api_rates?.success_per_sec ?? 0,
        failPerSec: s.api_rates?.failures_per_sec ?? 0,
        queue: s.redis?.queue_len ?? s.queue_remaining ?? 0,
        pending: s.redis?.stream_pending ?? s.stream_pending ?? 0,
        sold: s.mysql?.sold ?? 0,
      });
    }
    if (!pts.length) return;
    set({ chart: [...get().chart, ...pts].slice(-CHART_CAP) });
  },
  clearChart: () => set({ chart: [] }),
  mergeEvents: (ev) => {
    if (!ev?.length) return;
    const seen = new Set(get().events.map((e) => e.seq));
    const merged = [...get().events];
    for (const e of ev) {
      if (!seen.has(e.seq)) {
        merged.push(e);
        seen.add(e.seq);
      }
    }
    const trimmed = merged.length > 300 ? merged.slice(-300) : merged;
    set({ events: trimmed });
  },
}));
