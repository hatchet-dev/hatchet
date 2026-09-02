/**
 * Parse a Go-style duration string (e.g. "720h", "168h0m0s") into milliseconds.
 * Supports hours (h), minutes (m), and seconds (s).
 */
export function parseGoDuration(period: string): number | null {
  const re = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/;
  const match = period.trim().match(re);

  if (!match || (!match[1] && !match[2] && !match[3])) {
    return null;
  }

  const hours = parseInt(match[1] || '0', 10);
  const minutes = parseInt(match[2] || '0', 10);
  const seconds = parseInt(match[3] || '0', 10);

  return (hours * 3600 + minutes * 60 + seconds) * 1000;
}

/**
 * Returns the retention boundary Date (now - retentionPeriod).
 */
export function getRetentionBoundary(period: string): Date | null {
  const ms = parseGoDuration(period);
  if (ms === null) {
    return null;
  }
  return new Date(Date.now() - ms);
}

/**
 * Returns true if the given date falls before the retention boundary.
 * Uses a 5-minute tolerance so that filter windows matching the retention
 * period (e.g. "1 day" filter with 24h retention) don't falsely trigger.
 */
const RETENTION_TOLERANCE_MS = 5 * 60 * 1000;

export function isBeforeRetention(
  date: string | Date,
  period: string,
): boolean {
  const boundary = getRetentionBoundary(period);
  if (!boundary) {
    return false;
  }
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.getTime() < boundary.getTime() - RETENTION_TOLERANCE_MS;
}

/**
 * Formats a Go-style duration string into a human-readable label.
 * E.g. "720h" -> "30 days", "168h0m0s" -> "7 days", "24h" -> "1 day".
 */
export function formatRetentionPeriod(period: string): string {
  const ms = parseGoDuration(period);
  if (ms === null) {
    return period;
  }

  const hours = ms / (1000 * 3600);

  if (hours >= 24 && hours % 24 === 0) {
    const days = hours / 24;
    return days === 1 ? '1 day' : `${days} days`;
  }

  if (hours >= 1 && hours === Math.floor(hours)) {
    return hours === 1 ? '1 hour' : `${hours} hours`;
  }

  return period;
}
