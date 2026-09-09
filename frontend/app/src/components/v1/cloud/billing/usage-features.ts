import {
  OrganizationTenantResourceLimits,
  OrganizationUsageFeature,
  TenantResource,
  TenantResourceLimit,
} from '@/lib/api/generated/control-plane/data-contracts';

const PERIOD_USAGE_FEATURE_IDS = new Set(['task_runs', 'events']);

const DAILY_LIMIT_BY_PERIOD: Record<string, string> = {
  task_runs: 'task_runs_daily_limit',
  events: 'events_daily_limit',
};

const PERIOD_BY_DAILY_LIMIT: Record<string, string> = Object.fromEntries(
  Object.entries(DAILY_LIMIT_BY_PERIOD).map(([periodId, dailyId]) => [
    dailyId,
    periodId,
  ]),
);

const FEATURE_TO_RESOURCE: Record<string, TenantResource> = {
  task_runs: TenantResource.TASK_RUN,
  events: TenantResource.EVENT,
};

export const DAILY_LIMIT_WARN_PERCENT = 75;
export const DAILY_LIMIT_CRITICAL_PERCENT = 90;

export type DailyMeter = TenantResourceLimit & {
  tenantName?: string;
  showTenant?: boolean;
};

export type UsageDisplayRow = {
  feature: OrganizationUsageFeature;
  dailyMeter?: DailyMeter;
  showPeriodAsCount: boolean;
};

export type UsageSeverity = 'ok' | 'warn' | 'critical';

export function isPeriodUsageFeature(featureId: string) {
  return PERIOD_USAGE_FEATURE_IDS.has(featureId);
}

export function isDailyLimitFeature(featureId: string) {
  return featureId in PERIOD_BY_DAILY_LIMIT;
}

export function parseGoDurationMs(window?: string) {
  if (!window) {
    return null;
  }

  const match = window
    .trim()
    .match(/^(?:(\d+)h)?(?:(\d+)m)?(?:(\d+(?:\.\d+)?)s)?$/);
  if (!match || (!match[1] && !match[2] && !match[3])) {
    return null;
  }

  const hours = Number(match[1] ?? 0);
  const minutes = Number(match[2] ?? 0);
  const seconds = Number(match[3] ?? 0);
  const ms = ((hours * 60 + minutes) * 60 + seconds) * 1000;
  return ms > 0 ? ms : null;
}

export function nextRefillAt(
  meter: Pick<TenantResourceLimit, 'lastRefill' | 'window'>,
) {
  if (!meter.lastRefill) {
    return null;
  }
  const windowMs = parseGoDurationMs(meter.window);
  if (!windowMs) {
    return null;
  }
  const last = new Date(meter.lastRefill);
  if (Number.isNaN(last.getTime())) {
    return null;
  }
  return new Date(last.getTime() + windowMs);
}

export function formatTimeUntil(date: Date, now = Date.now()) {
  const ms = date.getTime() - now;
  if (ms <= 0) {
    return 'refilling soon';
  }

  const minutes = Math.round(ms / (1000 * 60));
  if (minutes < 60) {
    return `refills in ${Math.max(1, minutes)}m`;
  }

  const hours = Math.round(minutes / 60);
  if (hours < 48) {
    return `refills in ${hours}h`;
  }

  return `refills in ${Math.round(hours / 24)}d`;
}

export function meterWindowLabel(window?: string) {
  if (window === '24h0m0s' || window === '24h') {
    return 'Today';
  }
  if (window === '168h0m0s' || window === '168h') {
    return 'This week';
  }
  if (window === '720h0m0s' || window === '720h') {
    return 'This month';
  }
  return 'Current window';
}

export function meterPercent(
  meter: Pick<TenantResourceLimit, 'value' | 'limitValue'>,
) {
  if (meter.limitValue <= 0) {
    return 0;
  }
  return Math.min(100, (meter.value / meter.limitValue) * 100);
}

export function dailyMeterSeverity(
  meter: Pick<TenantResourceLimit, 'value' | 'limitValue' | 'alarmValue'>,
): UsageSeverity {
  if (meter.limitValue <= 0) {
    return 'ok';
  }

  const percent = meterPercent(meter);
  if (
    meter.value >= meter.limitValue ||
    percent >= DAILY_LIMIT_CRITICAL_PERCENT
  ) {
    return 'critical';
  }

  const alarmPercent = meter.alarmValue
    ? (meter.alarmValue / meter.limitValue) * 100
    : DAILY_LIMIT_WARN_PERCENT;
  if (percent >= DAILY_LIMIT_WARN_PERCENT || percent >= alarmPercent) {
    return 'warn';
  }

  return 'ok';
}

export function selectDailyMeters(
  tenants: OrganizationTenantResourceLimits[],
  tenantId: string,
): Partial<Record<string, DailyMeter>> {
  const selected =
    tenantId === 'all'
      ? tenants
      : tenants.filter((tenant) => tenant.tenantId === tenantId);

  const meters: Partial<Record<string, DailyMeter>> = {};

  for (const [featureId, resource] of Object.entries(FEATURE_TO_RESOURCE)) {
    const candidates: DailyMeter[] = [];
    for (const tenant of selected) {
      for (const limit of tenant.limits) {
        if (limit.resource !== resource || limit.limitValue <= 0) {
          continue;
        }
        candidates.push({
          ...limit,
          tenantName: tenant.tenantName,
          showTenant: tenantId === 'all' && selected.length > 1,
        });
      }
    }

    if (candidates.length === 0) {
      continue;
    }

    meters[featureId] = candidates.reduce((worst, next) =>
      meterPercent(next) > meterPercent(worst) ? next : worst,
    );
  }

  return meters;
}

export function toUsageDisplayRows(
  features: OrganizationUsageFeature[],
  dailyMeters: Partial<Record<string, DailyMeter>> = {},
): UsageDisplayRow[] {
  const byId = new Map(features.map((feature) => [feature.featureId, feature]));

  return features
    .filter((feature) => {
      const periodId = PERIOD_BY_DAILY_LIMIT[feature.featureId];
      return !periodId || !byId.has(periodId);
    })
    .map((feature) => ({
      feature,
      dailyMeter: dailyMeters[feature.featureId],
      showPeriodAsCount: isPeriodUsageFeature(feature.featureId),
    }));
}
