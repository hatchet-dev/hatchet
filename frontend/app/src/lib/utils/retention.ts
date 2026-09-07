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

export type TimeWindowPreset = '1h' | '6h' | '1d' | '7d';

export const TIME_WINDOW_HOURS: Record<TimeWindowPreset, number> = {
  '1h': 1,
  '6h': 6,
  '1d': 24,
  '7d': 168,
};

export const TIME_WINDOW_LABELS: Record<TimeWindowPreset, string> = {
  '1h': '1 hour',
  '6h': '6 hours',
  '1d': '1 day',
  '7d': '7 days',
};

const TIME_WINDOWS_DESC: TimeWindowPreset[] = ['7d', '1d', '6h', '1h'];

export function isTimeWindowPreset(value: string): value is TimeWindowPreset {
  return value in TIME_WINDOW_HOURS;
}

export function isTimeWindowOutsideRetention(
  timeWindow: string,
  period: string,
): boolean {
  if (!isTimeWindowPreset(timeWindow)) {
    return false;
  }

  const since = new Date(
    Date.now() - TIME_WINDOW_HOURS[timeWindow] * 60 * 60 * 1000,
  );
  return isBeforeRetention(since, period);
}

/** Largest preset that still fits inside the tenant retention window. */
export function largestAllowedTimeWindow(
  period?: string,
): TimeWindowPreset {
  if (!period) {
    return '7d';
  }

  for (const tw of TIME_WINDOWS_DESC) {
    if (!isTimeWindowOutsideRetention(tw, period)) {
      return tw;
    }
  }

  return '1h';
}

/**
 * Default picker window is 1 day, snapped down when the plan is shorter.
 */
export function defaultTimeWindowForRetention(
  period?: string,
): TimeWindowPreset {
  const max = largestAllowedTimeWindow(period);
  return TIME_WINDOW_HOURS[max] < 24 ? max : '1d';
}

export function formatShortDate(date: string | Date): string {
  const d = typeof date === 'string' ? new Date(date) : date;
  return d.toLocaleDateString(undefined, { month: 'short', day: 'numeric' });
}
