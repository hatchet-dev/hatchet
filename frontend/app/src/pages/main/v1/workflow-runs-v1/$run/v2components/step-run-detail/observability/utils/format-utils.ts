interface FormatDurationOpts {
  unit?: 'ns' | 'ms';
  precise?: boolean;
}

export function formatDuration(
  value: number,
  opts: FormatDurationOpts = {},
): string {
  const { unit = 'ms', precise = false } = opts;
  const ms = unit === 'ns' ? value / 1_000_000 : value;

  if (precise) {
    if (unit === 'ns' && ms < 1) {
      const us = value / 1_000;
      return us < 1 ? '<1µs' : `${Math.round(us)}µs`;
    }
    if (ms < 1) {
      return '<1ms';
    }
    if (ms < 1000) {
      return `${Math.round(ms)}ms`;
    }
    if (ms < 60_000) {
      return `${(ms / 1000).toFixed(2)}s`;
    }
    const m = Math.floor(ms / 60_000);
    const s = ((ms % 60_000) / 1000).toFixed(1);
    return `${m}m ${s}s`;
  }

  if (ms <= 0) {
    return '0s';
  }
  if (ms < 1) {
    return '<1ms';
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
  const pad = (n: number, len = 2) => String(n).padStart(len, '0');
  const date = `${d.getUTCFullYear()}-${pad(d.getUTCMonth() + 1)}-${pad(d.getUTCDate())}`;
  const time = `${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}`;
  if (opts?.ms) {
    return `${date} ${time}.${pad(d.getUTCMilliseconds(), 3)} UTC`;
  }
  return `${date} ${time} UTC`;
}
