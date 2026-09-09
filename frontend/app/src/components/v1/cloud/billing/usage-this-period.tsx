import { UpgradeGate, UpgradeGateDialog } from './upgrade-gate-dialog';
import {
  dailyMeterSeverity,
  formatTimeUntil,
  meterPercent,
  nextRefillAt,
  selectDailyMeters,
  toUsageDisplayRows,
  type DailyMeter,
  type UsageDisplayRow,
  type UsageSeverity,
} from './usage-features';
import { ZoomableChart } from '@/components/v1/molecules/charts/zoomable';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Skeleton } from '@/components/v1/ui/skeleton';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import { OrganizationUsageFeature } from '@/lib/api/generated/control-plane/data-contracts';
import { cn } from '@/lib/utils';
import { ArrowUpCircleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { format } from 'date-fns';
import { useEffect, useMemo, useState } from 'react';
import { Line, LineChart, ResponsiveContainer } from 'recharts';

const GRAPHABLE_FEATURES = new Set(['task_runs', 'events']);

type RangePreset = 'period' | '7d' | '30d';

function formatUsageCount(value: number) {
  return new Intl.NumberFormat('en-US').format(value);
}

function usagePercent(feature: OrganizationUsageFeature) {
  if (feature.unlimited || feature.includedUsage <= 0) {
    return 0;
  }
  return Math.min(100, (feature.usage / feature.includedUsage) * 100);
}

function usageSeverity(feature: OrganizationUsageFeature): UsageSeverity {
  if (feature.unlimited || feature.includedUsage <= 0) {
    return 'ok';
  }
  const percent = usagePercent(feature);
  if (percent > 95) {
    return 'critical';
  }
  if (percent > 75) {
    return 'warn';
  }
  return 'ok';
}

function gateForFeature(featureId: string): UpgradeGate {
  if (featureId === 'tenants') {
    return 'tenants';
  }
  if (featureId === 'users') {
    return 'users';
  }
  if (featureId === 'data_retention_days') {
    return 'retention';
  }
  return 'usage';
}

const severityStyles: Record<UsageSeverity, { value: string; bar: string }> = {
  ok: {
    value: 'text-muted-foreground',
    bar: 'bg-foreground',
  },
  warn: {
    value: 'text-yellow-500 dark:text-yellow-400',
    bar: 'bg-yellow-500 dark:bg-yellow-400',
  },
  critical: {
    value: 'text-red-500 dark:text-red-400',
    bar: 'bg-red-500 dark:bg-red-400',
  },
};

function formatUsageLabel(feature: OrganizationUsageFeature) {
  if (feature.unlimited) {
    return `${formatUsageCount(feature.usage)} / ∞`;
  }
  return `${formatUsageCount(feature.usage)} / ${formatUsageCount(feature.includedUsage)}`;
}

function formatPeriodCount(feature: OrganizationUsageFeature) {
  return `${formatUsageCount(feature.usage)} this period`;
}

function formatDailyMeterValue(meter: DailyMeter) {
  return `${formatUsageCount(meter.value)} / ${formatUsageCount(meter.limitValue)}`;
}

function dailyLimitKind(window?: string) {
  if (window === '24h0m0s' || window === '24h') {
    return 'daily limit';
  }
  if (window === '168h0m0s' || window === '168h') {
    return 'weekly limit';
  }
  if (window === '720h0m0s' || window === '720h') {
    return 'monthly limit';
  }
  return 'limit';
}

function DailyMeterAnnotation({ meter }: { meter: DailyMeter }) {
  const refill = nextRefillAt(meter);

  return (
    <span className="font-normal text-muted-foreground">
      {' '}
      ({dailyLimitKind(meter.window)}
      {refill ? (
        <>
          {' - '}
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger
                asChild
                onFocusCapture={(event) => {
                  event.stopPropagation();
                }}
              >
                <span
                  className="underline decoration-muted-foreground/50 decoration-dotted underline-offset-2"
                  onClick={(event) => {
                    event.stopPropagation();
                  }}
                >
                  {formatTimeUntil(refill)}
                </span>
              </TooltipTrigger>
              <TooltipContent side="top">
                {format(refill, 'yyyy-MM-dd HH:mm:ss.SSS zzz')}
              </TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </>
      ) : null}
      {meter.showTenant && meter.tenantName ? ` - ${meter.tenantName}` : null})
    </span>
  );
}

function formatRangeLabel(start?: string, end?: string) {
  if (!start || !end) {
    return null;
  }
  const startDate = new Date(start);
  const endDate = new Date(end);
  if (Number.isNaN(startDate.getTime()) || Number.isNaN(endDate.getTime())) {
    return null;
  }
  const opts: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
  };
  return `${startDate.toLocaleDateString('en-US', opts)} – ${endDate.toLocaleDateString('en-US', opts)}`;
}

function rangeForPreset(
  preset: RangePreset,
  periodStart?: string,
  periodEnd?: string,
) {
  const end = new Date();
  if (preset === 'period' && periodStart && periodEnd) {
    return { start: new Date(periodStart), end: new Date(periodEnd) };
  }
  const days = preset === '7d' ? 7 : 30;
  return {
    start: new Date(end.getTime() - days * 24 * 60 * 60 * 1000),
    end,
  };
}

function metricValue(
  featureId: string,
  point: { taskRuns: number; events: number },
) {
  return featureId === 'events' ? point.events : point.taskRuns;
}

function UsageSparkline({ values }: { values: number[] }) {
  const data = values.map((usage, index) => ({ index, usage }));
  const active = values.some((value) => value > 0);

  return (
    <div className="h-8 w-28 shrink-0">
      <ResponsiveContainer width="100%" height="100%">
        <LineChart
          data={data}
          margin={{ top: 4, right: 0, bottom: 4, left: 0 }}
        >
          <Line
            type="monotone"
            dataKey="usage"
            stroke={active ? '#34d399' : 'hsl(var(--muted-foreground))'}
            strokeWidth={1.5}
            dot={false}
            isAnimationActive={false}
          />
        </LineChart>
      </ResponsiveContainer>
    </div>
  );
}

function UsageMeter({
  row,
  sparkline,
  selectable,
  onSelect,
  onUpgrade,
}: {
  row: UsageDisplayRow;
  sparkline?: number[];
  selectable: boolean;
  onSelect: () => void;
  onUpgrade: () => void;
}) {
  const { feature, dailyMeter, showPeriodAsCount } = row;
  const percent = dailyMeter
    ? meterPercent(dailyMeter)
    : showPeriodAsCount
      ? 0
      : usagePercent(feature);
  const severity = dailyMeter
    ? dailyMeterSeverity(dailyMeter)
    : showPeriodAsCount
      ? 'ok'
      : usageSeverity(feature);
  const styles = severityStyles[severity];
  const showBar = dailyMeter
    ? dailyMeter.limitValue > 0
    : !showPeriodAsCount && !feature.unlimited && feature.includedUsage > 0;
  const upgradeLabel = dailyMeter
    ? `Upgrade to raise the ${feature.name} daily limit`
    : `Upgrade to raise the ${feature.name} limit`;
  const primaryValue = dailyMeter
    ? formatDailyMeterValue(dailyMeter)
    : showPeriodAsCount
      ? formatPeriodCount(feature)
      : formatUsageLabel(feature);

  const content = (
    <>
      <div className="min-w-0 flex-1 space-y-2">
        <div className="flex items-center justify-between gap-4">
          <div className="min-w-0">
            <p className="text-sm font-medium text-foreground">
              {feature.name}
            </p>
            {dailyMeter && showPeriodAsCount ? (
              <p className="text-xs text-muted-foreground">
                {formatPeriodCount(feature)}
              </p>
            ) : null}
          </div>
          <div className="flex items-center gap-1.5">
            <p
              className={cn(
                'text-sm tabular-nums',
                dailyMeter || !showPeriodAsCount
                  ? styles.value
                  : 'text-muted-foreground',
              )}
            >
              {primaryValue}
              {dailyMeter ? <DailyMeterAnnotation meter={dailyMeter} /> : null}
            </p>
            {severity !== 'ok' ? (
              <button
                type="button"
                onClick={(event) => {
                  event.stopPropagation();
                  onUpgrade();
                }}
                className={cn(
                  'rounded-sm p-0.5 transition-opacity hover:opacity-80',
                  styles.value,
                )}
                aria-label={upgradeLabel}
              >
                <ArrowUpCircleIcon className="h-3.5 w-3.5" />
              </button>
            ) : null}
          </div>
        </div>
        {showBar ? (
          <div className="h-1 overflow-hidden rounded-full bg-muted">
            <div
              className={cn('h-full rounded-full', styles.bar)}
              style={{ width: `${percent}%` }}
            />
          </div>
        ) : null}
      </div>
      {sparkline ? <UsageSparkline values={sparkline} /> : null}
    </>
  );

  if (selectable) {
    return (
      <div
        role="button"
        tabIndex={0}
        onClick={onSelect}
        onKeyDown={(event) => {
          if (event.key === 'Enter' || event.key === ' ') {
            event.preventDefault();
            onSelect();
          }
        }}
        className="flex w-full cursor-pointer items-center gap-4 px-3 py-3 text-left transition-colors hover:bg-muted/40"
      >
        {content}
      </div>
    );
  }

  return (
    <div className="flex w-full items-center gap-4 px-3 py-3">{content}</div>
  );
}

export function UsageThisPeriod({
  organizationId,
}: {
  organizationId?: string | null;
}) {
  const { canBill, isControlPlaneEnabled } = useControlPlane();
  const [detailFeatureId, setDetailFeatureId] = useState<string | null>(null);
  const [upgradeGate, setUpgradeGate] = useState<UpgradeGate | null>(null);
  const [rangePreset, setRangePreset] = useState<RangePreset>('period');
  const [tenantId, setTenantId] = useState('all');
  const [tenantOptions, setTenantOptions] = useState<
    { tenantId: string; tenantName: string }[]
  >([]);

  const usage = useQuery({
    ...queries.controlPlane.usage(organizationId ?? ''),
    enabled: isControlPlaneEnabled && canBill && !!organizationId,
  });

  const resourceLimits = useQuery({
    ...queries.controlPlane.tenantResourceLimits(organizationId ?? ''),
    refetchInterval: 2 * 60_000,
    enabled: isControlPlaneEnabled && canBill && !!organizationId,
  });

  const features = useMemo(
    () => [...(usage.data?.features ?? [])].sort((a, b) => b.usage - a.usage),
    [usage.data?.features],
  );
  const dailyMeters = useMemo(
    () => selectDailyMeters(resourceLimits.data?.tenants ?? [], tenantId),
    [resourceLimits.data?.tenants, tenantId],
  );
  const rows = useMemo(
    () => toUsageDisplayRows(features, dailyMeters),
    [dailyMeters, features],
  );
  const detailFeature = features.find(
    (feature) => feature.featureId === detailFeatureId,
  );

  const range = useMemo(
    () =>
      rangeForPreset(
        rangePreset,
        usage.data?.periodStart,
        usage.data?.periodEnd,
      ),
    [rangePreset, usage.data?.periodEnd, usage.data?.periodStart],
  );

  const timeseries = useQuery({
    ...queries.controlPlane.usageTimeseries(organizationId ?? '', {
      start: range.start.toISOString(),
      end: range.end.toISOString(),
      tenantId: tenantId === 'all' ? undefined : tenantId,
    }),
    enabled: isControlPlaneEnabled && canBill && !!organizationId,
  });

  useEffect(() => {
    if (tenantId !== 'all') {
      return;
    }

    const fromLimits = (resourceLimits.data?.tenants ?? []).map((tenant) => ({
      tenantId: tenant.tenantId,
      tenantName: tenant.tenantName,
    }));
    if (fromLimits.length > 0) {
      setTenantOptions(fromLimits);
      return;
    }
    if (!timeseries.data?.tenants) {
      return;
    }
    setTenantOptions(
      timeseries.data.tenants.map((tenant) => ({
        tenantId: tenant.tenantId,
        tenantName: tenant.tenantName,
      })),
    );
  }, [resourceLimits.data?.tenants, tenantId, timeseries.data?.tenants]);

  const seriesByFeature = useMemo(() => {
    const series = timeseries.data?.series ?? [];
    return {
      task_runs: series.map((point) => point.taskRuns),
      events: series.map((point) => point.events),
    };
  }, [timeseries.data?.series]);

  const chartData = useMemo(
    () =>
      (timeseries.data?.series ?? []).map((point) => ({
        date: `${point.date}T00:00:00.000Z`,
        usage: metricValue(detailFeatureId ?? 'task_runs', point),
      })),
    [detailFeatureId, timeseries.data?.series],
  );
  const chartTotal = chartData.reduce((sum, point) => sum + point.usage, 0);
  const tenantRows = timeseries.data?.tenants ?? [];
  const tenantTotal = tenantRows.reduce(
    (sum, tenant) => sum + metricValue(detailFeatureId ?? 'task_runs', tenant),
    0,
  );

  if (usage.isPending && !timeseries.data) {
    return (
      <Card
        variant="light"
        className="bg-transparent ring-1 ring-border/50 border-none"
      >
        <CardHeader className="p-4 border-b border-border/50">
          <Skeleton className="h-4 w-32" />
        </CardHeader>
        <CardContent className="space-y-3 p-4">
          <Skeleton className="h-6 w-full" />
          <Skeleton className="h-6 w-5/6" />
          <Skeleton className="h-6 w-full" />
        </CardContent>
      </Card>
    );
  }

  if (usage.isError && timeseries.isError) {
    return (
      <Alert variant="warn">
        <AlertTitle>Usage unavailable</AlertTitle>
        <AlertDescription className="flex flex-col gap-3">
          <span>We couldn&apos;t load usage for this billing period.</span>
          <div>
            <Button
              size="sm"
              variant="outline"
              onClick={() => {
                void usage.refetch();
                void timeseries.refetch();
                void resourceLimits.refetch();
              }}
            >
              Try again
            </Button>
          </div>
        </AlertDescription>
      </Alert>
    );
  }

  return (
    <>
      <Card
        variant="light"
        className="bg-transparent ring-1 ring-border/50 border-none"
      >
        <CardHeader className="p-4 border-b border-border/50">
          <div className="flex flex-col gap-3 lg:flex-row lg:items-center lg:justify-between">
            <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
              Usage
            </CardTitle>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              <Select
                value={rangePreset}
                onValueChange={(value) => setRangePreset(value as RangePreset)}
              >
                <SelectTrigger className="h-8 w-[170px] text-xs">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="period">
                    {formatRangeLabel(
                      usage.data?.periodStart,
                      usage.data?.periodEnd,
                    ) ?? 'Current period'}
                  </SelectItem>
                  <SelectItem value="7d">Last 7 days</SelectItem>
                  <SelectItem value="30d">Last 30 days</SelectItem>
                </SelectContent>
              </Select>
              <Select value={tenantId} onValueChange={setTenantId}>
                <SelectTrigger className="h-8 w-[180px] text-xs">
                  <SelectValue placeholder="All tenants" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">All tenants</SelectItem>
                  {tenantOptions.map((tenant) => (
                    <SelectItem key={tenant.tenantId} value={tenant.tenantId}>
                      {tenant.tenantName}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>
        </CardHeader>
        <CardContent className="p-0">
          {rows.length > 0 ? (
            <div className="divide-y divide-border/40">
              {rows.map((row) => {
                const graphable = GRAPHABLE_FEATURES.has(row.feature.featureId);
                const sparkline = graphable
                  ? seriesByFeature[
                      row.feature.featureId as keyof typeof seriesByFeature
                    ]
                  : undefined;

                return (
                  <UsageMeter
                    key={row.feature.featureId}
                    row={row}
                    sparkline={sparkline}
                    selectable={graphable}
                    onSelect={() => setDetailFeatureId(row.feature.featureId)}
                    onUpgrade={() =>
                      setUpgradeGate(gateForFeature(row.feature.featureId))
                    }
                  />
                );
              })}
            </div>
          ) : (
            <p className="p-4 text-sm text-muted-foreground">
              No usage recorded for this billing period.
            </p>
          )}
        </CardContent>
      </Card>

      {organizationId && upgradeGate ? (
        <UpgradeGateDialog
          open
          gate={upgradeGate}
          organizationId={organizationId}
          onDismiss={() => setUpgradeGate(null)}
        />
      ) : null}

      <Dialog
        open={!!detailFeatureId}
        onOpenChange={(open) => {
          if (!open) {
            setDetailFeatureId(null);
          }
        }}
      >
        <DialogContent className="max-w-3xl max-h-[85vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{detailFeature?.name ?? 'Usage'}</DialogTitle>
            <DialogDescription>
              Daily usage
              {formatRangeLabel(
                range.start.toISOString(),
                range.end.toISOString(),
              )
                ? ` · ${formatRangeLabel(range.start.toISOString(), range.end.toISOString())}`
                : ''}
              . Total {formatUsageCount(chartTotal)}.
            </DialogDescription>
          </DialogHeader>

          {timeseries.isError ? (
            <Alert variant="warn">
              <AlertTitle>Usage graph unavailable</AlertTitle>
              <AlertDescription>
                We couldn&apos;t load daily usage from your shards.
              </AlertDescription>
            </Alert>
          ) : timeseries.isPending ? (
            <Skeleton className="h-[220px] w-full" />
          ) : (
            <ZoomableChart<'usage'>
              kind="bar"
              showYAxis
              data={chartData}
              colors={{ usage: 'hsl(var(--foreground))' }}
              className="h-[240px] min-h-[240px]"
            />
          )}

          {tenantRows.length > 0 ? (
            <div className="space-y-3">
              <p className="text-sm font-medium text-foreground">
                Usage by tenant
              </p>
              <div className="divide-y divide-border/40">
                {tenantRows.map((tenant) => {
                  const value = metricValue(
                    detailFeatureId ?? 'task_runs',
                    tenant,
                  );
                  const percent =
                    tenantTotal > 0 ? (value / tenantTotal) * 100 : 0;
                  return (
                    <div
                      key={tenant.tenantId}
                      className="flex items-center justify-between gap-4 py-2"
                    >
                      <div>
                        <p className="text-sm text-foreground">
                          {tenant.tenantName}
                        </p>
                        <p className="text-xs text-muted-foreground">
                          {tenant.tenantSlug}
                        </p>
                      </div>
                      <p className="text-sm tabular-nums text-muted-foreground">
                        {formatUsageCount(value)}
                        {tenantTotal > 0 ? ` · ${percent.toFixed(1)}%` : ''}
                      </p>
                    </div>
                  );
                })}
              </div>
            </div>
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  );
}
