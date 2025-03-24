// Single unit durations
type SecondsDuration = `${number}s`;
type MinutesDuration = `${number}m`;
type HoursDuration = `${number}h`;
type DaysDuration = `${number}d`;

// Combined durations
type TwoUnitDurations =
  | `${number}d${number}h`
  | `${number}d${number}m`
  | `${number}d${number}s`
  | `${number}h${number}m`
  | `${number}h${number}s`
  | `${number}m${number}s`;
type ThreeUnitDurations =
  | `${number}d${number}h${number}m`
  | `${number}d${number}h${number}s`
  | `${number}d${number}m${number}s`
  | `${number}h${number}m${number}s`;
type FourUnitDuration = `${number}d${number}h${number}m${number}s`;

export type Duration =
  | SecondsDuration
  | MinutesDuration
  | HoursDuration
  | DaysDuration
  | TwoUnitDurations
  | ThreeUnitDurations
  | FourUnitDuration;
