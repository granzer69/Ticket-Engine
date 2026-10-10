import type { RunReport } from "../types";

/** Client poll deadline by preset (server may run longer). */
export function pollDeadlineMs(preset: string): number {
  const p = preset.toLowerCase();
  switch (p) {
    case "smoke":
    case "":
      return 2 * 60_000;
    case "claims":
      return 5 * 60_000;
    case "load-1k":
    case "1k":
      return 5 * 60_000;
    case "load-10k":
    case "10k":
      return 15 * 60_000;
    case "load-25k":
    case "25k":
    case "load-50k":
    case "50k":
      return 30 * 60_000;
    case "heavy-100k":
    case "100k":
      return 45 * 60_000;
    default:
      return 5 * 60_000;
  }
}

export function pollDeadlineLabel(preset: string): string {
  const ms = pollDeadlineMs(preset);
  const min = Math.round(ms / 60_000);
  return `${min}m`;
}

export async function pollRun(runId: string, preset: string): Promise<RunReport | null> {
  const deadline = Date.now() + pollDeadlineMs(preset);
  while (Date.now() < deadline) {
    const rep = await fetch(`/api/runs/${runId}`);
    const body = (await rep.json()) as RunReport;
    if (body.status !== "running") return body;
    await new Promise((r) => setTimeout(r, 500));
  }
  return null;
}
