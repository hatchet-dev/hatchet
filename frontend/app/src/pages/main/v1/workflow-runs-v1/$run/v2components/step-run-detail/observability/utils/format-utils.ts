interface FormatDurationOpts {
  unit?: "ns" | "ms";
  precise?: boolean;
}

export function formatDuration(
  value: number,
  opts: FormatDurationOpts = {},
): string {
  const { unit = "ms", precise = false } = opts;
  const ms = unit === "ns" ? value / 1_000_000 : value;

  if (precise) {
    if (unit === "ns" && ms < 1) {
      return `${(value / 1_000).toFixed(1)}µs`;
    }
    if (ms < 1) {
      return "<1ms";
    }
    if (ms < 1000) {
      return `${ms.toFixed(ms < 10 ? 2 : 1)}ms`;
    }
    if (ms < 60_000) {
      return `${(ms / 1000).toFixed(2)}s`;
    }
    const m = Math.floor(ms / 60_000);
    const s = ((ms % 60_000) / 1000).toFixed(1);
    return `${m}m ${s}s`;
  }

  if (ms <= 0) {
    return "0s";
  }
  if (ms < 1) {
    return "<1ms";
  }
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }
  if (ms < 60_000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  const m = Math.floor(ms / 60_000);
  const s = Math.floor((ms % 60_000) / 1000);
  return s > 0 ? `${m}m${s}s` : `${m}m`;
}

export function formatTimestamp(iso: string, opts?: { ms?: boolean }): string {
  const d = new Date(iso);
  const base = d.toLocaleString(undefined, {
    month: "short",
    day: "numeric",
    ...(opts?.ms ? {} : { year: "numeric" }),
    hour: "numeric",
    minute: "2-digit",
    second: "2-digit",
    hour12: true,
  });
  if (opts?.ms) {
    const millis = String(d.getMilliseconds()).padStart(3, "0");
    return `${base}.${millis}`;
  }
  return base;
}
