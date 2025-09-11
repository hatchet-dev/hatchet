// Single unit durations
type SecondsDuration = `${number}s`;
type MinutesDuration = `${number}m`;
type HoursDuration = `${number}h`;

// Combined durations
type TwoUnitDurations = `${number}h${number}m` | `${number}h${number}s` | `${number}m${number}s`;
type ThreeUnitDurations = `${number}h${number}m${number}s`;

export type Duration =
  | SecondsDuration
  | MinutesDuration
  | HoursDuration
  | TwoUnitDurations
  | ThreeUnitDurations;
