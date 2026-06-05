import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import {
  ChartConfig,
  ChartContainer,
  ChartTooltip,
  ChartTooltipContent,
} from '@/components/v1/ui/chart';
import { Spinner } from '@/components/v1/ui/loading';
import type { OrganizationUsageSummary } from '@/lib/api/generated/control-plane/data-contracts';
import { useMemo } from 'react';
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  XAxis,
  YAxis,
} from 'recharts';

// A small categorical palette cycled across tenants in the stacked chart.
const TENANT_COLORS = [
  'rgba(59, 130, 246, 0.85)',
  'rgba(34, 197, 94, 0.85)',
  'rgba(234, 179, 8, 0.85)',
  'rgba(168, 85, 247, 0.85)',
  'rgba(244, 114, 182, 0.85)',
  'rgba(14, 165, 233, 0.85)',
  'rgba(249, 115, 22, 0.85)',
  'rgba(20, 184, 166, 0.85)',
];

const numberFormatter = new Intl.NumberFormat();

// startOfUtcDay returns the epoch millis for the UTC midnight of the given date.
function startOfUtcDay(value: string): number {
  const date = new Date(value);
  return Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate());
}

// utcDayKey returns the YYYY-MM-DD UTC key used to join usage rows onto days.
function utcDayKey(value: string): string {
  return new Date(value).toISOString().slice(0, 10);
}

function formatDay(value: string): string {
  return new Date(value).toLocaleDateString([], {
    month: 'short',
    day: 'numeric',
  });
}

function formatRange(start: string, end: string): string {
  const opts: Intl.DateTimeFormatOptions = {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  };
  return `${new Date(start).toLocaleDateString([], opts)} – ${new Date(
    end,
  ).toLocaleDateString([], opts)}`;
}

interface UsageSummaryProps {
  summary?: OrganizationUsageSummary;
  isLoading: boolean;
}

export function UsageSummary({ summary, isLoading }: UsageSummaryProps) {
  const chartConfig = useMemo<ChartConfig>(() => {
    const config: ChartConfig = {};
    summary?.tenants.forEach((tenant, idx) => {
      config[tenant.tenantId] = {
        label: tenant.tenantName || tenant.tenantSlug || tenant.tenantId,
        color: TENANT_COLORS[idx % TENANT_COLORS.length],
      };
    });
    return config;
  }, [summary?.tenants]);

  // Pivot the daily per-tenant rows into one entry per day keyed by tenant id,
  // which is the shape the stacked BarChart expects. The backend only returns
  // rows for days with activity, so we zero-fill every day in the window
  // (dataStart..periodEnd) to render the full billing period.
  const chartData = useMemo(() => {
    if (!summary) {
      return [];
    }

    const usageByDay = new Map<string, Record<string, number>>();
    for (const point of summary.daily) {
      const key = utcDayKey(point.day);
      const existing = usageByDay.get(key) ?? {};
      existing[point.tenantId] =
        (existing[point.tenantId] ?? 0) + point.taskRunCount;
      usageByDay.set(key, existing);
    }

    const rows: Record<string, number | string>[] = [];
    const start = startOfUtcDay(summary.dataStart);
    const end = startOfUtcDay(summary.periodEnd);
    for (let ts = start; ts <= end; ts += 24 * 60 * 60 * 1000) {
      const iso = new Date(ts).toISOString();
      const key = iso.slice(0, 10);
      rows.push({ day: iso, ...(usageByDay.get(key) ?? {}) });
    }

    return rows;
  }, [summary]);

  const tenantIds = summary?.tenants.map((t) => t.tenantId) ?? [];

  // Line items for the breakdown panel. `cost` is intentionally left undefined
  // for now; a projected/actual $ value per metric will render here soon.
  const lineItems: {
    key: string;
    label: string;
    value: number;
    cost?: string;
  }[] = [
    {
      key: 'task_runs',
      label: 'Task runs',
      value: summary?.totalTaskRunCount ?? 0,
    },
    {
      key: 'events',
      label: 'Events',
      value: summary?.totalEventCount ?? 0,
    },
  ];

  return (
    <Card variant="light">
      <CardHeader>
        <CardTitle>Usage summary</CardTitle>
        {summary ? (
          <p className="mt-1 text-sm text-muted-foreground">
            Daily task runs by tenant for the current billing period (
            {formatRange(summary.periodStart, summary.periodEnd)}).
          </p>
        ) : null}
      </CardHeader>
      <CardContent>
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            {isLoading ? (
              <div className="flex h-64 items-center justify-center">
                <Spinner />
              </div>
            ) : chartData.length === 0 ? (
              <div className="flex h-64 items-center justify-center text-sm text-muted-foreground">
                No usage recorded for this billing period yet.
              </div>
            ) : (
              <ChartContainer config={chartConfig} className="h-64 w-full">
                <ResponsiveContainer width="100%" height="100%">
                  <BarChart
                    data={chartData}
                    margin={{ left: 0, right: 0, top: 8, bottom: 0 }}
                  >
                    <CartesianGrid vertical={false} />
                    <XAxis
                      dataKey="day"
                      tickFormatter={formatDay}
                      tickLine={false}
                      axisLine={false}
                      tickMargin={8}
                      minTickGap={16}
                      style={{ fontSize: '11px', userSelect: 'none' }}
                    />
                    <YAxis
                      tickLine={false}
                      axisLine={false}
                      width={48}
                      tickFormatter={(v) => numberFormatter.format(v as number)}
                      style={{ fontSize: '11px', userSelect: 'none' }}
                    />
                    <ChartTooltip
                      content={
                        <ChartTooltipContent
                          className="w-[200px] text-xs"
                          labelFormatter={(v) =>
                            new Date(v).toLocaleDateString()
                          }
                        />
                      }
                    />
                    {tenantIds.map((tenantId) => (
                      <Bar
                        key={tenantId}
                        dataKey={tenantId}
                        stackId="usage"
                        fill={chartConfig[tenantId]?.color}
                        isAnimationActive={false}
                      />
                    ))}
                  </BarChart>
                </ResponsiveContainer>
              </ChartContainer>
            )}
          </div>

          <div className="lg:col-span-1">
            <h4 className="font-mono text-xs font-normal uppercase tracking-wider text-muted-foreground">
              This period
            </h4>
            <div className="mt-2 divide-y divide-border/50">
              {lineItems.map((item) => (
                <div
                  key={item.key}
                  className="flex items-center justify-between py-3"
                >
                  <span className="text-sm text-muted-foreground">
                    {item.label}
                  </span>
                  <div className="flex items-baseline gap-3 text-right">
                    <span className="text-sm font-semibold tabular-nums text-foreground">
                      {numberFormatter.format(item.value)}
                    </span>
                    {item.cost ? (
                      <span className="w-16 text-sm tabular-nums text-muted-foreground">
                        {item.cost}
                      </span>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
