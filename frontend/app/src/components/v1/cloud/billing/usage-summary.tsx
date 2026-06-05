import { computeUsageSpend, type UsageSpendMetric } from './usage-spend';
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
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import type {
  OrganizationUsageSummary,
  SubscriptionPlan,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useMemo, type ReactNode } from 'react';
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

const currencyFormatter = new Intl.NumberFormat('en-US', {
  style: 'currency',
  currency: 'USD',
});

function formatCents(cents: number): string {
  return currencyFormatter.format(cents / 100);
}

function fmtInt(n: number): string {
  return numberFormatter.format(Math.round(n));
}

function formatPrice(dollars: number): string {
  return currencyFormatter.format(dollars);
}

// renderOverage shows a metric's overage cost for the current or projected
// column, or a muted "Included" when the usage stays within the plan allotment.
function renderOverage(
  metric: UsageSpendMetric | undefined,
  projected: boolean,
) {
  const cents = projected
    ? (metric?.projectedOverageCents ?? 0)
    : (metric?.overageCents ?? 0);

  if (metric?.metered && cents > 0) {
    return <span className="text-foreground">{formatCents(cents)}</span>;
  }
  return <span className="text-muted-foreground">Included</span>;
}

// AmountCell wraps an invoice amount with a hover tooltip explaining its math.
function AmountCell({
  className,
  amount,
  math,
}: {
  className: string;
  amount: ReactNode;
  math: ReactNode;
}) {
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span className={`cursor-help ${className}`}>{amount}</span>
      </TooltipTrigger>
      <TooltipContent className="max-w-[18rem] text-left">
        {math}
      </TooltipContent>
    </Tooltip>
  );
}

// overageMath explains how a metric's current or projected overage was derived.
function overageMath(
  label: string,
  unit: string,
  metric: UsageSpendMetric,
  projected: boolean,
  remainingDays: number,
  windowDays: number,
): ReactNode {
  if (!metric.metered) {
    return <p>{label} are included in your plan with no metered overage.</p>;
  }

  const usage = projected ? metric.projectedUsage : metric.usage;
  const overageUnits = projected
    ? metric.projectedOverageUnits
    : metric.overageUnits;
  const blocks = projected
    ? metric.projectedOverageBlocks
    : metric.overageBlocks;
  const cents = projected ? metric.projectedOverageCents : metric.overageCents;

  return (
    <div className="space-y-1.5">
      {projected ? (
        <p>
          Projected{' '}
          <span className="font-semibold">
            {fmtInt(usage)} {unit}
          </span>{' '}
          = {fmtInt(metric.usage)} so far + {fmtInt(metric.dailyRate)}/day ×{' '}
          {remainingDays.toFixed(1)} days left
          <span className="mt-0.5 block text-muted-foreground">
            daily rate averaged over the last {windowDays}{' '}
            {windowDays === 1 ? 'day' : 'days'}
          </span>
        </p>
      ) : (
        <p>
          <span className="font-semibold">{fmtInt(usage)}</span> {unit} used so
          far this period.
        </p>
      )}

      {overageUnits > 0 ? (
        <>
          <p>
            {fmtInt(usage)} − {fmtInt(metric.includedUsage)} included ={' '}
            {fmtInt(overageUnits)} over, billed in {fmtInt(metric.billingUnits)}{' '}
            blocks (rounded up to {fmtInt(blocks)}).
          </p>
          <p className="font-semibold">
            {fmtInt(blocks)} {blocks === 1 ? 'block' : 'blocks'} ×{' '}
            {formatPrice(metric.overagePrice)} = {formatCents(cents)}
          </p>
        </>
      ) : (
        <p>
          Within the {fmtInt(metric.includedUsage)} included {unit} — no
          overage.
        </p>
      )}
    </div>
  );
}

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
  // The organization's active plan, used to price current and projected spend.
  plan?: SubscriptionPlan;
  isLoading: boolean;
}

export function UsageSummary({ summary, plan, isLoading }: UsageSummaryProps) {
  const spend = useMemo(() => {
    if (!summary || !plan) {
      return undefined;
    }
    return computeUsageSpend(plan, summary);
  }, [summary, plan]);

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
    // Render the full billing period; future days (and inactive past days) are
    // zero-filled so the axis spans the whole period.
    const end = startOfUtcDay(summary.periodEnd);
    for (let ts = start; ts <= end; ts += 24 * 60 * 60 * 1000) {
      const iso = new Date(ts).toISOString();
      const key = iso.slice(0, 10);
      rows.push({ day: iso, ...(usageByDay.get(key) ?? {}) });
    }

    return rows;
  }, [summary]);

  const tenantIds = summary?.tenants.map((t) => t.tenantId) ?? [];

  // Invoice-style usage line items. When the plan is priced (spend present) the
  // amount column shows the metered overage; otherwise it falls back to raw
  // usage counts.
  const usageLines = [
    {
      key: 'task_runs',
      label: 'Task runs',
      unit: 'runs',
      value: summary?.totalTaskRunCount ?? 0,
      metric: spend?.taskRuns,
    },
    {
      key: 'events',
      label: 'Events',
      unit: 'events',
      value: summary?.totalEventCount ?? 0,
      metric: spend?.events,
    },
  ];

  const elapsedPct = spend ? Math.round(spend.fractionElapsed * 100) : 0;

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
            <TooltipProvider delayDuration={100}>
              <div className="mt-3">
                {spend ? (
                  <div className="flex items-center gap-3 pb-1.5">
                    <span className="flex-1" />
                    <span className="w-20 text-right font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                      Current
                    </span>
                    <span className="w-20 text-right font-mono text-[11px] uppercase tracking-wider text-muted-foreground">
                      Projected
                    </span>
                  </div>
                ) : null}

                {spend ? (
                  <div className="flex items-baseline gap-3 border-t border-border/50 py-3">
                    <div className="flex-1">
                      <div className="text-sm text-foreground">Base price</div>
                      {plan ? (
                        <div className="text-xs text-muted-foreground">
                          {plan.name}
                          {plan.period ? ` · ${plan.period}` : ''}
                        </div>
                      ) : null}
                    </div>
                    <AmountCell
                      className="w-20 text-right text-sm tabular-nums text-foreground"
                      amount={formatCents(spend.baseCents)}
                      math={
                        <p>
                          {plan?.name ?? 'Plan'} base price, charged each
                          billing period regardless of usage.
                        </p>
                      }
                    />
                    <AmountCell
                      className="w-20 text-right text-sm tabular-nums text-muted-foreground"
                      amount={formatCents(spend.baseCents)}
                      math={
                        <p>
                          {plan?.name ?? 'Plan'} base price, charged each
                          billing period regardless of usage.
                        </p>
                      }
                    />
                  </div>
                ) : null}

                {usageLines.map((line) => (
                  <div
                    key={line.key}
                    className="flex items-baseline gap-3 border-t border-border/50 py-3"
                  >
                    <div className="flex-1">
                      <div className="text-sm text-foreground">
                        {line.label}
                      </div>
                      <div className="text-xs tabular-nums text-muted-foreground">
                        {numberFormatter.format(line.value)} {line.unit}
                      </div>
                    </div>
                    {spend && line.metric ? (
                      <>
                        <AmountCell
                          className="w-20 text-right text-sm tabular-nums"
                          amount={renderOverage(line.metric, false)}
                          math={overageMath(
                            line.label,
                            line.unit,
                            line.metric,
                            false,
                            spend.remainingDays,
                            spend.windowDays,
                          )}
                        />
                        <AmountCell
                          className="w-20 text-right text-sm tabular-nums"
                          amount={renderOverage(line.metric, true)}
                          math={overageMath(
                            line.label,
                            line.unit,
                            line.metric,
                            true,
                            spend.remainingDays,
                            spend.windowDays,
                          )}
                        />
                      </>
                    ) : (
                      <span className="text-sm font-semibold tabular-nums text-foreground">
                        {numberFormatter.format(line.value)}
                      </span>
                    )}
                  </div>
                ))}

                {spend ? (
                  <div className="flex items-baseline gap-3 border-t border-border/50 py-3">
                    <span className="flex-1 text-sm font-semibold text-foreground">
                      Total
                    </span>
                    <AmountCell
                      className="w-20 text-right text-base font-semibold tabular-nums text-foreground"
                      amount={formatCents(spend.currentCents)}
                      math={
                        <p>
                          Base {formatCents(spend.baseCents)} + task runs{' '}
                          {formatCents(spend.taskRuns.overageCents)} + events{' '}
                          {formatCents(spend.events.overageCents)} ={' '}
                          {formatCents(spend.currentCents)} accrued so far (
                          {elapsedPct}% of the period elapsed).
                        </p>
                      }
                    />
                    <AmountCell
                      className="w-20 text-right text-base font-medium tabular-nums text-muted-foreground"
                      amount={formatCents(spend.projectedCents)}
                      math={
                        <p>
                          Base {formatCents(spend.baseCents)} + task runs{' '}
                          {formatCents(spend.taskRuns.projectedOverageCents)} +
                          events{' '}
                          {formatCents(spend.events.projectedOverageCents)} ={' '}
                          {formatCents(spend.projectedCents)}, projecting
                          current usage across the full period.
                        </p>
                      }
                    />
                  </div>
                ) : null}
              </div>
            </TooltipProvider>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
