import { useEffect } from "react";
import type { Snapshot } from "../types";
import { useLabStore } from "../store/labStore";

export function useTelemetrySSE() {
  const setSnap = useLabStore((s) => s.setSnap);
  const setConnected = useLabStore((s) => s.setConnected);
  const pushChart = useLabStore((s) => s.pushChartFromSnap);
  const mergeEvents = useLabStore((s) => s.mergeEvents);

  useEffect(() => {
    fetch("/api/snapshot")
      .then((r) => (r.ok ? r.json() : null))
      .then((snap) => {
        if (snap) {
          setSnap(snap as Snapshot);
          pushChart(snap as Snapshot);
        }
      })
      .catch(() => {});

    const es = new EventSource("/api/stream");
    es.onopen = () => setConnected(true);
    es.onerror = () => setConnected(false);
    es.addEventListener("snapshot", (ev) => {
      try {
        const snap = JSON.parse(ev.data) as Snapshot;
        setSnap(snap);
        pushChart(snap);
        if (snap.events?.length) mergeEvents(snap.events);
      } catch {
        /* ignore */
      }
    });
    return () => {
      es.close();
      setConnected(false);
    };
  }, [mergeEvents, pushChart, setConnected, setSnap]);
}
