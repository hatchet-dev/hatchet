import { Duration } from 'date-fns';

export function formatDuration(duration: Duration, rawTimeMs: number): string {
  const parts = [];

  if (duration.days) {
    parts.push(`${duration.days}d`);
  }

  if (duration.hours) {
    parts.push(`${duration.hours}h`);
  }

  if (duration.minutes) {
    parts.push(`${duration.minutes}m`);
  }

  if (rawTimeMs < 10000 && duration.seconds) {
    const ms = Math.floor((rawTimeMs % 1000) / 10);
    parts.push(`${duration.seconds}.${ms.toString().padStart(2, '0')}s`);
    return parts.join(' ');
  }

  if (duration.seconds) {
    parts.push(`${duration.seconds}s`);
  }

  if (rawTimeMs < 1000) {
    const ms = rawTimeMs % 1000;
    parts.push(`${ms}ms`);
  }

  return parts.join(' ');
}
