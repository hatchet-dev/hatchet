import type {
  SubscriptionPlan,
  SubscriptionPlanFeature,
} from '@/lib/api/generated/control-plane/data-contracts';

// Autumn feature ids for the metered usage we bill on. These match
// EventNameTaskRuns / EventNameEvents on the control plane.
export const TASK_RUNS_FEATURE_ID = 'task_runs';
export const EVENTS_FEATURE_ID = 'events';

const DAY_MS = 24 * 60 * 60 * 1000;

interface UsageRecentWindow {
  start: string;
  end: string;
  taskRunCount: number;
  eventCount: number;
}

export interface UsageSpendInput {
  periodStart: string;
  periodEnd: string;
  totalTaskRunCount: number;
  totalEventCount: number;
  // A trailing usage window (independent of the billing period) used to derive
  // the daily rate for projecting spend across the remaining days.
  recent: UsageRecentWindow;
}

// UsageSpendMetric is the priced breakdown for a single metered feature,
// carrying enough detail to explain how each amount was derived.
export interface UsageSpendMetric {
  // Total usage so far this period.
  usage: number;
  // Usage extrapolated to the full period (current + dailyRate * remainingDays).
  projectedUsage: number;
  // Average daily usage from the trailing window, used to project.
  dailyRate: number;
  // Usage included in the plan before overage applies.
  includedUsage: number;
  // Units per overage price increment (e.g. 1,000,000).
  billingUnits: number;
  // Overage price (dollars) charged per billingUnits.
  overagePrice: number;
  // Units over the included amount (current / projected).
  overageUnits: number;
  projectedOverageUnits: number;
  // Whole billing blocks charged for the overage (current / projected).
  overageBlocks: number;
  projectedOverageBlocks: number;
  // Overage cost in cents (current / projected); 0 within the included amount.
  overageCents: number;
  projectedOverageCents: number;
  // Whether the plan meters this feature (has overage pricing). When false the
  // feature is included/unlimited and never accrues overage.
  metered: boolean;
}

export interface UsageSpend {
  baseCents: number;
  taskRuns: UsageSpendMetric;
  events: UsageSpendMetric;
  currentCents: number;
  projectedCents: number;
  // Fraction of the billing period elapsed (0..1), shown in the explainer.
  fractionElapsed: number;
  // Number of trailing days the projection's daily rate is averaged over.
  windowDays: number;
  // Days remaining in the billing period used to extrapolate usage.
  remainingDays: number;
}

// findPlanFeature locates a feature by id across the plan's feature groups.
export function findPlanFeature(
  plan: SubscriptionPlan | undefined,
  featureId: string,
): SubscriptionPlanFeature | undefined {
  for (const group of plan?.featureGroups ?? []) {
    const feature = group.features.find((f) => f.featureId === featureId);
    if (feature) {
      return feature;
    }
  }
  return undefined;
}

// overageBlocks returns the number of whole billing blocks (e.g. 1M-unit
// blocks) charged for usage beyond the included amount. Overage is billed in
// whole blocks, so any partial block rounds up.
export function overageBlocks(
  usage: number,
  feature: SubscriptionPlanFeature | undefined,
): number {
  if (!feature || feature.unlimited || !feature.overage) {
    return 0;
  }
  const { billingUnits } = feature.overage;
  if (!billingUnits) {
    return 0;
  }
  const overageUnits = Math.max(0, usage - (feature.includedUsage ?? 0));
  if (overageUnits <= 0) {
    return 0;
  }
  return Math.ceil(overageUnits / billingUnits);
}

// overageCents returns the metered overage cost (in cents) for usage beyond the
// feature's included amount. Charged per whole billing block.
function overageCents(
  usage: number,
  feature: SubscriptionPlanFeature | undefined,
): number {
  const blocks = overageBlocks(usage, feature);
  if (blocks <= 0 || !feature?.overage) {
    return 0;
  }
  return Math.round(blocks * feature.overage.price * 100);
}

// fractionElapsed returns how far through the billing period we are (0..1).
export function fractionElapsed(
  periodStart: string,
  periodEnd: string,
  nowMs: number = Date.now(),
): number {
  const start = new Date(periodStart).getTime();
  const end = new Date(periodEnd).getTime();
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    return 1;
  }
  const fraction = (nowMs - start) / (end - start);
  return Math.min(1, Math.max(fraction, 0));
}

// trailingDailyRates converts the trailing usage window into an average daily
// rate per metric. The window is a fixed span of complete UTC days fetched
// independently of the billing period, so it works even early in a new period.
function trailingDailyRates(input: UsageSpendInput): {
  taskRunsPerDay: number;
  eventsPerDay: number;
  windowDays: number;
} {
  const start = new Date(input.recent.start).getTime();
  const end = new Date(input.recent.end).getTime();
  if (!Number.isFinite(start) || !Number.isFinite(end) || end <= start) {
    return { taskRunsPerDay: 0, eventsPerDay: 0, windowDays: 0 };
  }

  const windowDays = (end - start) / DAY_MS;
  if (windowDays <= 0) {
    return { taskRunsPerDay: 0, eventsPerDay: 0, windowDays: 0 };
  }

  return {
    taskRunsPerDay: input.recent.taskRunCount / windowDays,
    eventsPerDay: input.recent.eventCount / windowDays,
    windowDays: Math.round(windowDays),
  };
}

// computeUsageSpend derives current and projected spend from the active plan:
// the plan base price plus metered overage on task runs and events above each
// feature's included amount. Projection holds the usage accrued so far and adds
// the trailing daily rate applied across the days remaining in the period.
export function computeUsageSpend(
  plan: SubscriptionPlan | undefined,
  input: UsageSpendInput,
  nowMs: number = Date.now(),
): UsageSpend {
  const baseCents = plan?.amountCents ?? 0;
  const taskFeature = findPlanFeature(plan, TASK_RUNS_FEATURE_ID);
  const eventFeature = findPlanFeature(plan, EVENTS_FEATURE_ID);

  const { taskRunsPerDay, eventsPerDay, windowDays } =
    trailingDailyRates(input);

  const periodEndMs = new Date(input.periodEnd).getTime();
  const remainingDays = Number.isFinite(periodEndMs)
    ? Math.max(0, (periodEndMs - nowMs) / DAY_MS)
    : 0;

  const buildMetric = (
    usage: number,
    dailyRate: number,
    feature: SubscriptionPlanFeature | undefined,
  ): UsageSpendMetric => {
    const metered = !!feature && !feature.unlimited && !!feature.overage;
    const includedUsage = feature?.includedUsage ?? 0;
    const billingUnits = feature?.overage?.billingUnits ?? 0;
    const overagePrice = feature?.overage?.price ?? 0;
    const projectedUsage = usage + dailyRate * remainingDays;

    return {
      usage,
      projectedUsage,
      dailyRate,
      includedUsage,
      billingUnits,
      overagePrice,
      overageUnits: metered ? Math.max(0, usage - includedUsage) : 0,
      projectedOverageUnits: metered
        ? Math.max(0, projectedUsage - includedUsage)
        : 0,
      overageBlocks: overageBlocks(usage, feature),
      projectedOverageBlocks: overageBlocks(projectedUsage, feature),
      overageCents: overageCents(usage, feature),
      projectedOverageCents: overageCents(projectedUsage, feature),
      metered,
    };
  };

  const taskRuns = buildMetric(
    input.totalTaskRunCount,
    taskRunsPerDay,
    taskFeature,
  );
  const events = buildMetric(input.totalEventCount, eventsPerDay, eventFeature);

  const currentCents = baseCents + taskRuns.overageCents + events.overageCents;
  const projectedCents =
    baseCents + taskRuns.projectedOverageCents + events.projectedOverageCents;

  return {
    baseCents,
    taskRuns,
    events,
    currentCents,
    projectedCents,
    fractionElapsed: fractionElapsed(input.periodStart, input.periodEnd, nowMs),
    windowDays,
    remainingDays,
  };
}
