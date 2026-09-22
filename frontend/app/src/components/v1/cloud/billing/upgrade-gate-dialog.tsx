import { payAsYouGoPlan, planCodeBase } from './subscription-plan-code';
import {
  setupCardDialogClassName,
  SetupCard,
} from '@/components/layout/setup-card';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
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
import { formatRetentionPeriod } from '@/lib/utils/retention';
import { ChevronDownIcon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
import { useState } from 'react';

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
  retentionPeriod?: string;
};

// ---------------------------------------------------------------------------
// Copy. Keys mirror docs/plans/upgrade-gate-copy.mdx; edit there, then here.
// ---------------------------------------------------------------------------

const COPY = {
  planName: 'Pay as you Go',
  freeName: 'Free',
  header: {
    tenants: {
      title: "You've hit your tenant limit",
      description:
        'Upgrade to pay-as-you-go to add more tenants and separate dev, staging, and prod.',
    },
    users: {
      title: "You've hit your member limit",
      description: 'Upgrade to pay-as-you-go to bring in your whole team.',
    },
    retention: {
      title: "You've gone past your retention window",
      description: 'Upgrade to pay-as-you-go to unlock longer retention.',
    },
    usageResource: {
      title: (resource: string) =>
        `You've hit your limit for ${resource.toLowerCase()}`,
      description: (resource: string) =>
        `Upgrade to pay-as-you-go to unlock more ${resource.toLowerCase()}.`,
    },
    usageGeneric: {
      title: 'Upgrade to Pay as you Go',
      description: 'Upgrade to pay-as-you-go to remove the free tier limits.',
    },
  },
  compare: {
    resource: 'Resource',
    perDay: '/ day',
    perMonth: '/ month',
    unlimited: 'Unlimited',
    expand: 'See full breakdown',
    collapse: 'Hide full breakdown',
  },
  price: {
    label: 'Base price',
    free: '$0',
    payg: '$0, then usage-based',
  },
  actions: {
    upgrade: 'Upgrade',
    dismiss: 'Not now',
    footnote:
      'No monthly fee. You pay nothing until you scale past the included usage.',
    pricing: 'Pricing details',
    salesLead: 'Need volume discounts or custom terms?',
    sales: 'Talk to us',
  },
  error: {
    title: "We couldn't change your plan",
  },
} as const;

// Row order in the comparison table. `label` is user-facing.
const RESOURCES = [
  { id: 'tenants', label: 'Tenants' },
  { id: 'users', label: 'Members' },
  { id: 'data_retention_days', label: 'Data retention' },
  { id: 'task_runs', label: 'Task runs' },
  { id: 'events', label: 'Events' },
  { id: 'worker_slots_limit', label: 'Concurrent runs' },
  { id: 'crons', label: 'Crons' },
  { id: 'scheduled_runs', label: 'Scheduled runs' },
  { id: 'webhooks', label: 'Webhook endpoints' },
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
): GateHeader {
  switch (gate) {
    case 'tenants':
      return COPY.header.tenants;
    case 'users':
      return COPY.header.users;
    case 'retention':
      return COPY.header.retention;
    case 'usage':
    default:
      if (row) {
        return {
          title: COPY.header.usageResource.title(row.label),
          description: COPY.header.usageResource.description(row.label),
        };
      }
      return COPY.header.usageGeneric;
  }
}

// ---------------------------------------------------------------------------
// Hook: everything the dialog and the inline card need
// ---------------------------------------------------------------------------

function useUpgradeGate({
  gate,
  organizationId,
  featureId,
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
    header: buildHeader(gate, highlighted),
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

function ComparisonTable({ comparison }: { comparison: Comparison }) {
  const highlighted = comparison.rows.find(
    (row) => row.id === comparison.highlightId,
  );
  // With no gated resource there is nothing to collapse to, so show it all.
  const [expanded, setExpanded] = useState(!highlighted);
  const rows = expanded ? comparison.rows : highlighted ? [highlighted] : [];

  return (
    <div className="flex flex-col gap-2">
      <div className="overflow-hidden rounded-lg border border-border/50">
        <Table>
          <TableHeader>
            <TableRow className="hover:bg-transparent">
              <TableHead className="h-9 px-3 text-xs">
                {COPY.compare.resource}
              </TableHead>
              <TableHead className="h-9 px-3 text-right text-xs">
                {COPY.freeName}
              </TableHead>
              <TableHead className="h-9 px-3 text-right text-xs text-foreground">
                {COPY.planName}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {rows.map((row) => {
              const isGated = row.id === comparison.highlightId;
              return (
                <TableRow
                  key={row.id}
                  className={cn(isGated && 'bg-primary/5 hover:bg-primary/5')}
                >
                  <TableCell
                    className={cn(
                      'px-3 py-2 text-sm',
                      isGated
                        ? 'font-medium text-foreground'
                        : 'text-muted-foreground',
                    )}
                  >
                    {row.label}
                  </TableCell>
                  <TableCell className="px-3 py-2 text-right text-sm tabular-nums text-muted-foreground">
                    {row.free}
                  </TableCell>
                  <TableCell className="px-3 py-2 text-right text-sm font-medium tabular-nums text-foreground">
                    {row.payg}
                  </TableCell>
                </TableRow>
              );
            })}
            {expanded ? (
              <TableRow className="bg-muted/30 hover:bg-muted/30">
                <TableCell className="px-3 py-2 text-sm text-muted-foreground">
                  {COPY.price.label}
                </TableCell>
                <TableCell className="px-3 py-2 text-right text-sm tabular-nums text-muted-foreground">
                  {COPY.price.free}
                </TableCell>
                <TableCell className="px-3 py-2 text-right text-sm font-medium text-foreground">
                  {COPY.price.payg}
                </TableCell>
              </TableRow>
            ) : null}
          </TableBody>
        </Table>
      </div>
      {highlighted ? (
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="w-fit gap-1 px-2 text-xs text-muted-foreground"
          aria-expanded={expanded}
          onClick={() => setExpanded((value) => !value)}
        >
          {expanded ? COPY.compare.collapse : COPY.compare.expand}
          <ChevronDownIcon
            className={cn(
              'size-3 transition-transform',
              expanded && 'rotate-180',
            )}
          />
        </Button>
      ) : null}
    </div>
  );
}

function UpgradeGateBody({ state }: { state: UpgradeGateState }) {
  const { comparison, upgrade, salesHref } = state;

  return (
    <div className="flex flex-col gap-4">
      <ComparisonTable comparison={comparison} />

      {upgrade.isError ? (
        <Alert variant="destructive">
          <AlertTitle>{COPY.error.title}</AlertTitle>
          <AlertDescription>
            {getPlanChangeErrorMessage(upgrade.error)}
          </AlertDescription>
        </Alert>
      ) : null}

      <p className="text-xs text-muted-foreground">
        {COPY.actions.footnote}{' '}
        <a
          href={PRICING_URL}
          target="_blank"
          rel="noreferrer"
          className="underline underline-offset-4 hover:text-foreground"
        >
          {COPY.actions.pricing}
        </a>
        {' · '}
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
  );
}

function UpgradeGateFooter({
  state,
  onDismiss,
}: {
  state: UpgradeGateState;
  onDismiss?: () => void;
}) {
  const { upgrade, canUpgrade, onUpgrade } = state;
  return (
    <>
      {onDismiss ? (
        <Button type="button" variant="outline" size="sm" onClick={onDismiss}>
          {COPY.actions.dismiss}
        </Button>
      ) : null}
      <Button
        type="button"
        size="sm"
        disabled={!canUpgrade || upgrade.isPending}
        onClick={onUpgrade}
      >
        {upgrade.isPending ? <Spinner /> : COPY.actions.upgrade}
      </Button>
    </>
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
      footer={<UpgradeGateFooter state={state} onDismiss={onDismiss} />}
    >
      <UpgradeGateBody state={state} />
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
      <DialogContent
        className={`${setupCardDialogClassName} max-h-[85vh] max-w-xl overflow-y-auto`}
      >
        <DialogTitle className="sr-only">{state.header.title}</DialogTitle>
        <SetupCard
          className="max-w-none"
          title={state.header.title}
          description={
            <DialogDescription>{state.header.description}</DialogDescription>
          }
          footer={<UpgradeGateFooter state={state} onDismiss={onDismiss} />}
        >
          <UpgradeGateBody state={state} />
        </SetupCard>
      </DialogContent>
    </Dialog>
  );
}
