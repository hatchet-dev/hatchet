import { payAsYouGoPlan, planCodeBase } from './subscription-plan-code';
import { SetupCard } from '@/components/layout/setup-card';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizationEntitlements } from '@/hooks/use-organization-entitlements';
import type { RetentionAttempt } from '@/hooks/use-retention-gate';
import {
  getPlanChangeErrorMessage,
  useSubscriptionUpgrade,
} from '@/hooks/use-subscription-upgrade';
import { useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import {
  OrganizationResourceLimit,
  SubscriptionPlan,
  SubscriptionPlanFeature,
  SubscriptionPlanList,
} from '@/lib/api/generated/control-plane/data-contracts';
import { OFFICE_HOURS_URL, PRICING_URL } from '@/lib/external-links';
import { cn } from '@/lib/utils';
import {
  TIME_WINDOW_LABELS,
  formatRetentionPeriod,
  formatShortDate,
} from '@/lib/utils/retention';
import {
  BoltIcon,
  BuildingOffice2Icon,
  CalendarDaysIcon,
  ClockIcon,
  GlobeAltIcon,
  UsersIcon,
} from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';

export type UpgradeGate = 'tenants' | 'users' | 'retention' | 'usage';

export type UpgradeGateProps = {
  gate: UpgradeGate;
  organizationId: string;
  onDismiss?: () => void;
  /**
   * Feature that hit its limit (e.g. `task_runs`, `crons`). Names the resource
   * in the header and highlights its row for the `usage` gate. Omit for a
   * generic "upgrade" entry point with no specific limit.
   */
  featureId?: string;
  retentionAttempt?: RetentionAttempt | null;
  retentionPeriod?: string;
};

// ---------------------------------------------------------------------------
// Copy. Keys mirror docs/plans/upgrade-gate-copy.mdx; edit there, then here.
// ---------------------------------------------------------------------------

const COPY = {
  planName: 'Pay as you Go',
  freeName: 'Free',
  callout: {
    limitReached: 'Free tier limit.',
    usedOf: (used: string, limit: string) => `${used} of ${limit} used`,
    included: (value: string) => `${value} included`,
  },
  header: {
    tenants: {
      title: 'Tenant limit reached',
      description: (free: string, payg: string) =>
        `The Free tier includes ${free} tenant. Pay as you Go includes ${payg}, so you can separate dev, staging, and prod.`,
    },
    users: {
      title: 'Member limit reached',
      description: (free: string, payg: string) =>
        `The Free tier includes ${free} members. Pay as you Go includes ${payg}, so your whole team can work in Hatchet.`,
    },
    retention: {
      title: 'Outside your retention window',
      triedPreset: (window: string) => `You tried to view the last ${window}.`,
      triedSince: (date: string) => `You tried to look back to ${date}.`,
      description: (free: string, payg: string) =>
        `The Free tier keeps ${free} of runs, events, and logs. Pay as you Go keeps ${payg}.`,
    },
    usageResource: {
      title: (resource: string) => `${resource} limit reached`,
      description: (free: string, payg: string) =>
        `The Free tier includes ${free}. Pay as you Go includes ${payg}, and you only pay for usage beyond that.`,
    },
    usageGeneric: {
      title: 'Upgrade to Pay as you Go',
      description:
        'Pay as you Go removes the Free tier limits. There is no monthly fee, and you pay nothing until you scale past what is included.',
    },
  },
  detail: {
    tenants: (payg: string) => `Pay as you Go includes ${payg} tenants.`,
    users: (payg: string) => `Pay as you Go includes ${payg} members.`,
    retention: (payg: string) => `Pay as you Go keeps ${payg} of history.`,
    usage: (payg: string) => `Pay as you Go includes ${payg}.`,
  },
  compare: {
    resource: 'Resource',
    perDay: '/ day',
    perMonth: '/ month',
    unlimited: 'Unlimited',
  },
  price: {
    label: 'Base price',
    free: '$0',
    payg: '$0, then usage-based',
  },
  actions: {
    upgrade: 'Upgrade to Pay as you Go',
    dismiss: 'Not now',
    footnote:
      'No monthly fee. You pay nothing until you scale past the included usage.',
    pricing: 'Pricing details',
    salesLead: 'Need volume discounts or custom terms?',
    sales: 'Talk to us',
  },
  error: {
    title: 'Plan change failed',
  },
} as const;

// Row order in the comparison table. `label` is user-facing.
const RESOURCES = [
  { id: 'tenants', label: 'Tenants', icon: BuildingOffice2Icon },
  { id: 'users', label: 'Members', icon: UsersIcon },
  { id: 'data_retention_days', label: 'Data retention', icon: ClockIcon },
  { id: 'task_runs', label: 'Task runs', icon: BoltIcon },
  { id: 'events', label: 'Events', icon: BoltIcon },
  { id: 'worker_slots_limit', label: 'Concurrent runs', icon: BoltIcon },
  { id: 'crons', label: 'Crons', icon: CalendarDaysIcon },
  { id: 'scheduled_runs', label: 'Scheduled runs', icon: CalendarDaysIcon },
  { id: 'webhooks', label: 'Webhook endpoints', icon: GlobeAltIcon },
] as const;

type ResourceId = (typeof RESOURCES)[number]['id'];

// Only shown when the plans API is unavailable (e.g. billing disabled).
const PAYG_FALLBACK: Record<ResourceId, string> = {
  tenants: '10',
  users: '50',
  data_retention_days: '7',
  task_runs: '1M',
  events: '10M',
  worker_slots_limit: '500',
  crons: '1,000',
  scheduled_runs: '1M',
  webhooks: '100',
};

// Only shown when the plans API is unavailable. Tenants, members, and
// retention prefer the org's live entitlements; task runs and events prefer
// the plan list's daily free limits.
const FREE_FALLBACK: Record<ResourceId, string> = {
  tenants: '1',
  users: '3',
  data_retention_days: '1',
  task_runs: '1,000',
  events: '10,000',
  worker_slots_limit: '100',
  crons: '5',
  scheduled_runs: '100',
  webhooks: '5',
};

const DAILY_LIMIT_TO_RESOURCE: Record<string, ResourceId> = {
  task_runs_daily_limit: 'task_runs',
  events_daily_limit: 'events',
};

type ComparisonRow = {
  id: ResourceId;
  label: string;
  free: string;
  payg: string;
};

type Comparison = {
  rows: ComparisonRow[];
  highlightId: ResourceId | null;
};

type GateHeader = {
  title: string;
  description: string;
};

type LimitCallout = {
  icon: typeof BuildingOffice2Icon;
  label: string;
  value: string;
  detail: string;
};

type Entitlements = {
  tenants?: OrganizationResourceLimit;
  users?: OrganizationResourceLimit;
};

// ---------------------------------------------------------------------------
// Data helpers
// ---------------------------------------------------------------------------

function findFeature(plan: SubscriptionPlan | undefined, featureId: string) {
  return plan?.featureGroups
    ?.flatMap((group) => group.features)
    .find((feature) => feature.featureId === featureId);
}

function formatCount(value: number) {
  if (value >= 1_000_000 && value % 1_000_000 === 0) {
    return `${value / 1_000_000}M`;
  }
  return new Intl.NumberFormat('en-US').format(value);
}

function formatIncluded(feature: SubscriptionPlanFeature | undefined) {
  if (!feature || !feature.included) {
    return null;
  }
  if (feature.unlimited) {
    return COPY.compare.unlimited;
  }
  return formatCount(feature.includedUsage);
}

function formatResourceLimit(limit: OrganizationResourceLimit | undefined) {
  if (!limit) {
    return null;
  }
  if (limit.unlimited || limit.limit < 0) {
    return COPY.compare.unlimited;
  }
  return formatCount(limit.limit);
}

function formatDays(value: string) {
  if (value === COPY.compare.unlimited) {
    return value;
  }
  return value === '1' ? '1 day' : `${value} days`;
}

function freePlan(plans?: SubscriptionPlan[]) {
  return plans?.find((plan) => planCodeBase(plan.planCode) === 'free');
}

function toResourceId(featureId?: string): ResourceId | null {
  if (!featureId) {
    return null;
  }
  const mapped = DAILY_LIMIT_TO_RESOURCE[featureId] ?? featureId;
  return RESOURCES.some((resource) => resource.id === mapped)
    ? (mapped as ResourceId)
    : null;
}

function salesUrl(tenantName?: string, tenantId?: string) {
  if (!tenantName && !tenantId) {
    return OFFICE_HOURS_URL;
  }
  return `https://cal.com/team/hatchet/website-demo?notes=${encodeURIComponent(
    `Custom pricing request for tenant '${tenantName || 'Unknown'}' (${tenantId ?? ''})`,
  )}`;
}

function buildComparison(input: {
  planList?: SubscriptionPlanList;
  entitlements?: Entitlements;
  retentionPeriod?: string;
  highlightId: ResourceId | null;
}): Comparison {
  const payg = payAsYouGoPlan(input.planList?.plans);
  const free = freePlan(input.planList?.plans);
  const freeDaily = new Map(
    (input.planList?.freeLimits ?? []).map((limit) => [
      DAILY_LIMIT_TO_RESOURCE[limit.featureId] ?? limit.featureId,
      formatCount(limit.limit),
    ]),
  );

  const paygValue = (id: ResourceId) =>
    formatIncluded(findFeature(payg, id)) ?? PAYG_FALLBACK[id];

  const freeValue = (id: ResourceId) => {
    const fromPlan = formatIncluded(findFeature(free, id));
    switch (id) {
      case 'tenants':
        return (
          formatResourceLimit(input.entitlements?.tenants) ??
          fromPlan ??
          FREE_FALLBACK[id]
        );
      case 'users':
        return (
          formatResourceLimit(input.entitlements?.users) ??
          fromPlan ??
          FREE_FALLBACK[id]
        );
      case 'task_runs':
      case 'events':
        return freeDaily.get(id) ?? fromPlan ?? FREE_FALLBACK[id];
      default:
        return fromPlan ?? FREE_FALLBACK[id];
    }
  };

  const rows = RESOURCES.map((resource): ComparisonRow => {
    const id = resource.id;
    let free = freeValue(id);
    let payg = paygValue(id);

    if (id === 'data_retention_days') {
      free = input.retentionPeriod
        ? formatRetentionPeriod(input.retentionPeriod)
        : formatDays(free);
      payg = formatDays(payg);
    }

    if (id === 'task_runs' || id === 'events') {
      free = `${free} ${COPY.compare.perDay}`;
      payg = `${payg} ${COPY.compare.perMonth}`;
    }

    return { id, label: resource.label, free, payg };
  });

  return { rows, highlightId: input.highlightId };
}

function buildHeader(
  gate: UpgradeGate,
  row: ComparisonRow | undefined,
  retentionAttempt?: RetentionAttempt | null,
): GateHeader {
  switch (gate) {
    case 'tenants':
      return {
        title: COPY.header.tenants.title,
        description: COPY.header.tenants.description(
          row?.free ?? FREE_FALLBACK.tenants,
          row?.payg ?? PAYG_FALLBACK.tenants,
        ),
      };
    case 'users':
      return {
        title: COPY.header.users.title,
        description: COPY.header.users.description(
          row?.free ?? FREE_FALLBACK.users,
          row?.payg ?? PAYG_FALLBACK.users,
        ),
      };
    case 'retention': {
      const tried = retentionAttempt
        ? retentionAttempt.kind === 'preset'
          ? COPY.header.retention.triedPreset(
              TIME_WINDOW_LABELS[retentionAttempt.window],
            )
          : COPY.header.retention.triedSince(
              formatShortDate(retentionAttempt.date),
            )
        : null;
      const description = COPY.header.retention.description(
        row?.free ?? formatDays(FREE_FALLBACK.data_retention_days),
        row?.payg ?? formatDays(PAYG_FALLBACK.data_retention_days),
      );
      return {
        title: COPY.header.retention.title,
        description: tried ? `${tried} ${description}` : description,
      };
    }
    case 'usage':
    default:
      if (row) {
        return {
          title: COPY.header.usageResource.title(row.label),
          description: COPY.header.usageResource.description(
            row.free,
            row.payg,
          ),
        };
      }
      return {
        title: COPY.header.usageGeneric.title,
        description: COPY.header.usageGeneric.description,
      };
  }
}

function buildCallout(
  gate: UpgradeGate,
  row: ComparisonRow | undefined,
  entitlements?: Entitlements,
): LimitCallout | null {
  if (!row) {
    return null;
  }
  const icon =
    RESOURCES.find((resource) => resource.id === row.id)?.icon ?? BoltIcon;

  const usedOf = (limit?: OrganizationResourceLimit) =>
    limit && !limit.unlimited && limit.limit >= 0
      ? COPY.callout.usedOf(formatCount(limit.used), formatCount(limit.limit))
      : COPY.callout.included(row.free);

  switch (gate) {
    case 'tenants':
      return {
        icon,
        label: row.label,
        value: usedOf(entitlements?.tenants),
        detail: COPY.detail.tenants(row.payg),
      };
    case 'users':
      return {
        icon,
        label: row.label,
        value: usedOf(entitlements?.users),
        detail: COPY.detail.users(row.payg),
      };
    case 'retention':
      return {
        icon,
        label: row.label,
        value: row.free,
        detail: COPY.detail.retention(row.payg),
      };
    case 'usage':
    default:
      return {
        icon,
        label: row.label,
        value: COPY.callout.included(row.free),
        detail: COPY.detail.usage(row.payg),
      };
  }
}

// ---------------------------------------------------------------------------
// Hook: everything the dialog and the inline card need
// ---------------------------------------------------------------------------

function useUpgradeGate({
  gate,
  organizationId,
  featureId,
  retentionAttempt,
  retentionPeriod,
}: Omit<UpgradeGateProps, 'onDismiss'>) {
  const { canBill, isControlPlaneEnabled } = useControlPlane();
  const { tenant } = useTenantDetails();
  const { entitlements } = useOrganizationEntitlements(organizationId);
  const plansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: isControlPlaneEnabled && canBill,
  });
  const payg = payAsYouGoPlan(plansQuery.data?.plans);
  const upgrade = useSubscriptionUpgrade(organizationId);

  const highlightId: ResourceId | null =
    gate === 'tenants'
      ? 'tenants'
      : gate === 'users'
        ? 'users'
        : gate === 'retention'
          ? 'data_retention_days'
          : toResourceId(featureId);

  const comparison = buildComparison({
    planList: plansQuery.data,
    entitlements,
    retentionPeriod,
    highlightId,
  });
  const highlighted = comparison.rows.find((row) => row.id === highlightId);

  return {
    header: buildHeader(gate, highlighted, retentionAttempt),
    callout: buildCallout(gate, highlighted, entitlements),
    comparison,
    upgrade,
    canUpgrade: isControlPlaneEnabled && canBill && !!payg,
    onUpgrade: () => payg && upgrade.mutate(payg.planCode),
    salesHref: salesUrl(tenant?.name, tenant?.metadata?.id),
  };
}

type UpgradeGateState = ReturnType<typeof useUpgradeGate>;

// ---------------------------------------------------------------------------
// Presentation
// ---------------------------------------------------------------------------

function LimitCalloutBox({ callout }: { callout: LimitCallout }) {
  const Icon = callout.icon;
  return (
    <div className="flex items-start gap-3 rounded-lg border border-primary/40 bg-primary/5 p-4">
      <Icon className="mt-0.5 h-5 w-5 shrink-0 text-foreground" />
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-baseline justify-between gap-x-3 gap-y-1">
          <p className="text-sm font-medium text-foreground">{callout.label}</p>
          <p className="text-sm font-semibold tabular-nums text-foreground">
            {callout.value}
          </p>
        </div>
        <p className="mt-1 text-sm text-muted-foreground">
          <span className="font-medium text-foreground">
            {COPY.callout.limitReached}
          </span>{' '}
          {callout.detail}
        </p>
      </div>
    </div>
  );
}

function ComparisonTable({ comparison }: { comparison: Comparison }) {
  return (
    <div className="overflow-hidden rounded-lg border border-border">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead className="px-3">{COPY.compare.resource}</TableHead>
            <TableHead className="px-3 text-right">{COPY.freeName}</TableHead>
            <TableHead className="px-3 text-right text-foreground">
              {COPY.planName}
            </TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {comparison.rows.map((row) => {
            const highlighted = row.id === comparison.highlightId;
            return (
              <TableRow
                key={row.id}
                className={cn(highlighted && 'bg-primary/5 hover:bg-primary/5')}
              >
                <TableCell
                  className={cn(
                    'px-3 py-2',
                    highlighted
                      ? 'font-medium text-foreground'
                      : 'text-muted-foreground',
                  )}
                >
                  {row.label}
                </TableCell>
                <TableCell className="px-3 py-2 text-right tabular-nums text-muted-foreground">
                  {row.free}
                </TableCell>
                <TableCell className="px-3 py-2 text-right font-medium tabular-nums text-foreground">
                  {row.payg}
                </TableCell>
              </TableRow>
            );
          })}
          <TableRow className="bg-muted/30 hover:bg-muted/30">
            <TableCell className="px-3 py-2 text-muted-foreground">
              {COPY.price.label}
            </TableCell>
            <TableCell className="px-3 py-2 text-right tabular-nums text-muted-foreground">
              {COPY.price.free}
            </TableCell>
            <TableCell className="px-3 py-2 text-right font-medium text-foreground">
              {COPY.price.payg}
            </TableCell>
          </TableRow>
        </TableBody>
      </Table>
    </div>
  );
}

function UpgradeGateBody({
  state,
  onDismiss,
}: {
  state: UpgradeGateState;
  onDismiss?: () => void;
}) {
  const { callout, comparison, upgrade, canUpgrade, onUpgrade, salesHref } =
    state;

  return (
    <div className="flex flex-col gap-5">
      {callout ? <LimitCalloutBox callout={callout} /> : null}

      <ComparisonTable comparison={comparison} />

      {upgrade.isError ? (
        <Alert variant="destructive">
          <AlertTitle>{COPY.error.title}</AlertTitle>
          <AlertDescription>
            {getPlanChangeErrorMessage(upgrade.error)}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="flex flex-col gap-3">
        <DialogFooter className="gap-2 sm:gap-2">
          {onDismiss ? (
            <Button type="button" variant="outline" onClick={onDismiss}>
              {COPY.actions.dismiss}
            </Button>
          ) : null}
          <Button
            type="button"
            disabled={!canUpgrade || upgrade.isPending}
            onClick={onUpgrade}
          >
            {upgrade.isPending ? <Spinner /> : COPY.actions.upgrade}
          </Button>
        </DialogFooter>
        <p className="text-center text-xs text-muted-foreground sm:text-right">
          {COPY.actions.footnote}{' '}
          <a
            href={PRICING_URL}
            target="_blank"
            rel="noreferrer"
            className="underline underline-offset-4 hover:text-foreground"
          >
            {COPY.actions.pricing}
          </a>
        </p>
        <p className="text-center text-xs text-muted-foreground sm:text-right">
          {COPY.actions.salesLead}{' '}
          <a
            href={salesHref}
            target="_blank"
            rel="noreferrer"
            className="underline underline-offset-4 hover:text-foreground"
          >
            {COPY.actions.sales}
          </a>
        </p>
      </div>
    </div>
  );
}

/**
 * Inline (non-dialog) variant, used where the gate replaces a full-screen
 * form such as tenant creation.
 */
export function UpgradeGateContent({ onDismiss, ...props }: UpgradeGateProps) {
  const state = useUpgradeGate(props);

  return (
    <SetupCard
      className="max-w-none"
      title={state.header.title}
      description={state.header.description}
    >
      <UpgradeGateBody state={state} onDismiss={onDismiss} />
    </SetupCard>
  );
}

export function UpgradeGateDialog({
  open,
  onDismiss,
  ...props
}: UpgradeGateProps & { open: boolean; onDismiss: () => void }) {
  const state = useUpgradeGate(props);

  return (
    <Dialog open={open} onOpenChange={(next) => !next && onDismiss()}>
      <DialogContent className="max-h-[85vh] max-w-xl overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{state.header.title}</DialogTitle>
          <DialogDescription>{state.header.description}</DialogDescription>
        </DialogHeader>
        <UpgradeGateBody state={state} onDismiss={onDismiss} />
      </DialogContent>
    </Dialog>
  );
}
