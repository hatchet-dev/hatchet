// Single unit durations
type MillisecondsDuration = `${number}ms`;
type SecondsDuration = `${number}s`;
type MinutesDuration = `${number}m`;
type HoursDuration = `${number}h`;

// Combined durations
type TwoUnitDurations =
  | `${number}h${number}m`
  | `${number}h${number}s`
  | `${number}h${number}ms`
  | `${number}m${number}s`
  | `${number}m${number}ms`
  | `${number}s${number}ms`;
type ThreeUnitDurations =
  | `${number}h${number}m${number}s`
  | `${number}h${number}m${number}ms`
  | `${number}h${number}s${number}ms`
  | `${number}m${number}s${number}ms`;
type FourUnitDurations = `${number}h${number}m${number}s${number}ms`;

export type Duration =
  | MillisecondsDuration
  | SecondsDuration
  | MinutesDuration
  | HoursDuration
  | TwoUnitDurations
  | ThreeUnitDurations
  | FourUnitDurations;

export function durationToMilliseconds(duration: Duration): number {
  // Supports Go-style duration strings limited to h/m/s as defined by the Duration type.
  // Examples: "10s", "1m", "1m5s", "2h10m30s"
  const re = /(\d+)(ms|h|m|s)/g;

  let total = 0;
  let matched = false;

  for (let m = re.exec(duration); m !== null; m = re.exec(duration)) {
    matched = true;
    const value = Number(m[1]);
    const unit = m[2];

    if (!Number.isFinite(value) || value < 0) {
      throw new Error(`Invalid duration value: '${duration}'`);
    }

    switch (unit) {
      case 'h':
        total += value * 60 * 60 * 1000;
        break;
      case 'm':
        total += value * 60 * 1000;
        break;
      case 's':
        total += value * 1000;
        break;
      case 'ms':
        total += value;
        break;
      default:
        // should be unreachable due to regex
        throw new Error(`Invalid duration unit: '${duration}'`);
    }
  }

  if (!matched || total <= 0) {
    throw new Error(`Invalid duration: '${duration}'`);
  }

  return total;
}
