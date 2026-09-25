export type ResourceLimitStatus = 'ok' | 'warn' | 'exhausted';

export const USAGE_LIMIT_WARN_PERCENT = 75;

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

export const getUsageLimitStatus = ({
  usage,
  includedUsage,
  unlimited,
}: {
  usage: number;
  includedUsage: number;
  unlimited: boolean;
}): ResourceLimitStatus => {
  if (unlimited || includedUsage <= 0) {
    return 'ok';
  }

  if (usage >= includedUsage) {
    return 'exhausted';
  }

  if ((usage / includedUsage) * 100 > USAGE_LIMIT_WARN_PERCENT) {
    return 'warn';
  }

  return 'ok';
};
