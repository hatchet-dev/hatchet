type SecondsDuration = `${number}s`;
type MinutesDuration = `${number}m`;
type HoursDuration = `${number}h`;
type TwoUnitDurations = `${number}h${number}m` | `${number}h${number}s` | `${number}m${number}s`;
type ThreeUnitDurations = `${number}h${number}m${number}s`;

type DurationString =
  | SecondsDuration
  | MinutesDuration
  | HoursDuration
  | TwoUnitDurations
  | ThreeUnitDurations;

export interface DurationObject {
  hours?: number;
  minutes?: number;
  seconds?: number;
}

/** A number is treated as milliseconds. */
export type Duration = DurationString | DurationObject | number;

const DURATION_RE = /^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+)s)?$/;

/** Normalizes a Duration to Go-style string format (e.g. "1h30m5s"). */
export function durationToString(d: Duration): string {
  if (typeof d === 'string') {
    return d;
  }
  if (typeof d === 'number') {
    const totalSeconds = Math.floor(d / 1000);
    const h = Math.floor(totalSeconds / 3600);
    const m = Math.floor((totalSeconds % 3600) / 60);
    const s = totalSeconds % 60;
    let out = '';
    if (h) {
      out += `${h}h`;
    }
    if (m) {
      out += `${m}m`;
    }
    if (s || !out) {
      out += `${s}s`;
    }
    return out;
  }
  let s = '';
  if (d.hours) {
    s += `${d.hours}h`;
  }
  if (d.minutes) {
    s += `${d.minutes}m`;
  }
  if (d.seconds) {
    s += `${d.seconds}s`;
  }
  return s || '0s';
}

export function durationToMs(d: Duration): number {
  if (typeof d === 'number') {
    return d;
  }
  if (typeof d === 'object') {
    return ((d.hours ?? 0) * 3600 + (d.minutes ?? 0) * 60 + (d.seconds ?? 0)) * 1000;
  }

  const match = (d as string).match(DURATION_RE);
  if (!match) {
    throw new Error(
      `Invalid duration string: "${d}". Expected format like "1h30m5s", "10m", "30s".`
    );
  }

  const [, h, m, s] = match;
  return (
    (parseInt(h ?? '0', 10) * 3600 + parseInt(m ?? '0', 10) * 60 + parseInt(s ?? '0', 10)) * 1000
  );
}
