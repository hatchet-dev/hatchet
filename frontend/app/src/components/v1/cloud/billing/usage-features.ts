import {
  OrganizationTenantResourceLimits,
  OrganizationUsageFeature,
  TenantResource,
  TenantResourceLimit,
} from '@/lib/api/generated/control-plane/data-contracts';

const PERIOD_USAGE_FEATURE_IDS = new Set(['task_runs', 'events']);

// Plan allowances we do not meter. Showing them as 0 / limit reads as usage.
const HIDDEN_USAGE_FEATURE_IDS = new Set([
  'throughput_rps',
  'active_storage_gb',
  'network_bandwidth_gb',
  'data_retention_days',
  'managed_compute_cents',
  'queue_backlog_limit',
  'ingested_bytes',
]);

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
  // Live inventory from the shard. Not a billing-period total and not a plan allowance.
  countOnly?: boolean;
};

export type ShardUsageCounts = {
  taskRuns: number;
  events: number;
  crons?: number;
  scheduledRuns?: number;
  webhooks?: number;
  tenants?: { used: number; limit: number; unlimited: boolean };
  users?: { used: number; limit: number; unlimited: boolean };
};

const SHARD_USAGE_ROWS: {
  id: string;
  name: string;
  kind: 'period' | 'inventory' | 'allowance';
}[] = [
  { id: 'task_runs', name: 'Task Runs', kind: 'period' },
  { id: 'events', name: 'Events', kind: 'period' },
  { id: 'crons', name: 'Crons', kind: 'inventory' },
  { id: 'scheduled_runs', name: 'Scheduled Runs', kind: 'inventory' },
  { id: 'webhooks', name: 'Webhook Endpoints', kind: 'inventory' },
  { id: 'tenants', name: 'Tenants', kind: 'allowance' },
  { id: 'users', name: 'Users', kind: 'allowance' },
];

function shardCount(counts: ShardUsageCounts, id: string): number | undefined {
  switch (id) {
    case 'task_runs':
      return counts.taskRuns;
    case 'events':
      return counts.events;
    case 'crons':
      return counts.crons;
    case 'scheduled_runs':
      return counts.scheduledRuns;
    case 'webhooks':
      return counts.webhooks;
    case 'tenants':
      return counts.tenants?.used;
    case 'users':
      return counts.users?.used;
    default:
      return undefined;
  }
}

export function shardUsageRows(
  counts: ShardUsageCounts,
  dailyMeters: Partial<Record<string, DailyMeter>> = {},
): UsageDisplayRow[] {
  const rows: UsageDisplayRow[] = [];

  for (const definition of SHARD_USAGE_ROWS) {
    const value = shardCount(counts, definition.id);
    if (value === undefined) {
      continue;
    }

    const allowance =
      definition.id === 'tenants'
        ? counts.tenants
        : definition.id === 'users'
          ? counts.users
          : undefined;

    rows.push({
      feature: {
        featureId: definition.id,
        name: definition.name,
        usage: value,
        includedUsage: allowance?.unlimited ? 0 : (allowance?.limit ?? 0),
        unlimited: allowance?.unlimited ?? false,
      },
      dailyMeter: dailyMeters[definition.id],
      showPeriodAsCount: definition.kind === 'period',
      countOnly: definition.kind === 'inventory',
    });
  }

  return rows;
}

export type UsageSeverity = 'ok' | 'warn' | 'critical';

export type UsageRangePreset = 'period' | '7d' | '30d';

export function isPeriodUsageFeature(featureId: string) {
  return PERIOD_USAGE_FEATURE_IDS.has(featureId);
}

export function sumUsageSeries(values: number[]) {
  return values.reduce((sum, value) => sum + value, 0);
}

export function formatObservedUsage(count: number, preset: UsageRangePreset) {
  const formatted = new Intl.NumberFormat('en-US').format(count);
  if (preset === '7d') {
    return `${formatted} in the last 7 days`;
  }
  if (preset === '30d') {
    return `${formatted} in the last 30 days`;
  }
  return `${formatted} this month`;
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
    .filter((feature) => !HIDDEN_USAGE_FEATURE_IDS.has(feature.featureId))
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

export const TENANT_USAGE_COLORS = [
  'hsl(221 83% 60%)',
  'hsl(160 72% 45%)',
  'hsl(38 92% 55%)',
  'hsl(270 70% 66%)',
  'hsl(350 78% 62%)',
  'hsl(188 80% 48%)',
  'hsl(25 90% 58%)',
  'hsl(82 65% 48%)',
];

export type TenantDailyPoint = {
  date: string;
  taskRuns: number;
  events: number;
};

export type TenantUsageBreakdown = {
  tenantId: string;
  tenantName: string;
  series?: TenantDailyPoint[];
};

export type TenantChartSeries = {
  tenantId: string;
  name: string;
  color: string;
};

export function tenantUsageColor(
  tenants: { tenantId: string }[],
  tenantId: string,
) {
  const index = tenants.findIndex((tenant) => tenant.tenantId === tenantId);
  const slot = index < 0 ? 0 : index;
  return TENANT_USAGE_COLORS[slot % TENANT_USAGE_COLORS.length];
}

export function tenantUsageChart(
  tenants: TenantUsageBreakdown[],
  featureId: string,
  selectedTenantId?: string | null,
) {
  if (!tenants.some((tenant) => (tenant.series?.length ?? 0) > 0)) {
    return null;
  }

  const colored = tenants.map((tenant) => ({
    tenant,
    color: tenantUsageColor(tenants, tenant.tenantId),
  }));
  const selected = colored.filter(
    (entry) => entry.tenant.tenantId === selectedTenantId,
  );
  const shown = selected.length > 0 ? selected : colored;
  const axis =
    colored.find((entry) => (entry.tenant.series?.length ?? 0) > 0)?.tenant
      .series ?? [];

  let total = 0;
  const points = axis.map((point) => {
    const row: { date: string } & Record<string, number> = {
      date: point.date.includes('T')
        ? point.date
        : `${point.date}T00:00:00.000Z`,
    };
    for (const entry of shown) {
      const day = entry.tenant.series?.find((item) => item.date === point.date);
      const value =
        featureId === 'events' ? (day?.events ?? 0) : (day?.taskRuns ?? 0);
      row[entry.tenant.tenantId] = value;
      total += value;
    }
    return row;
  });

  return {
    points,
    series: shown.map((entry) => ({
      tenantId: entry.tenant.tenantId,
      name: entry.tenant.tenantName,
      color: entry.color,
    })),
    total,
  };
}
