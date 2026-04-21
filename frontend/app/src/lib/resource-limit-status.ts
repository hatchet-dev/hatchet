export type ResourceLimitStatus = 'ok' | 'warn' | 'exhausted';

export const getResourceLimitStatus = ({
  value,
  alarmValue,
  limitValue,
}: {
  value: number;
  alarmValue?: number;
  limitValue: number;
}): ResourceLimitStatus => {
  if (value >= limitValue) {
    return 'exhausted';
  }

  if (alarmValue && value >= alarmValue) {
    return 'warn';
  }

  return 'ok';
};
