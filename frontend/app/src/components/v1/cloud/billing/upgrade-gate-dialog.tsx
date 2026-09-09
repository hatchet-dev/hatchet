import { payAsYouGoPlan } from './subscription-plan-code';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading';
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
  SubscriptionPlan,
  SubscriptionPlanFeature,
} from '@/lib/api/generated/control-plane/data-contracts';
import { OFFICE_HOURS_URL, PRICING_URL } from '@/lib/external-links';
import { cn } from '@/lib/utils';
import { formatRetentionPeriod } from '@/lib/utils/retention';
import {
  BoltIcon,
  BuildingOffice2Icon,
  ClockIcon,
  CurrencyDollarIcon,
} from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';

export type UpgradeGate = 'tenants' | 'users' | 'retention' | 'usage';

const PAYG_SUBHEAD =
  'Pay as you Go removes the free tier limits; you pay nothing until you scale past them.';
const DISMISS_LABEL = 'Not now';

type UpgradeGateProps = {
  gate: UpgradeGate;
  organizationId: string;
  onDismiss?: () => void;
  retentionAttempt?: RetentionAttempt | null;
  retentionPeriod?: string;
};

type Benefit = {
  id: UpgradeGate;
  title: string;
  description: string;
  icon: typeof BuildingOffice2Icon;
};

const PAYG_FEATURES = {
  tenants: { id: 'tenants', fallback: '10' },
  users: { id: 'users', fallback: '50' },
  retentionDays: { id: 'data_retention_days', fallback: '7' },
  concurrent: { id: 'worker_slots_limit', fallback: '500' },
  scheduled: { id: 'scheduled_runs', fallback: '1M' },
  crons: { id: 'crons', fallback: '1,000' },
  webhooks: { id: 'webhooks', fallback: '100' },
  taskRuns: { id: 'task_runs', fallback: '1M' },
  events: { id: 'events', fallback: '10M' },
} as const;

function findFeature(plan: SubscriptionPlan | undefined, featureId: string) {
  return plan?.featureGroups
    ?.flatMap((group) => group.features)
    .find((feature) => feature.featureId === featureId);
}

function formatIncluded(feature?: SubscriptionPlanFeature, fallback = '—') {
  if (!feature) {
    return fallback;
  }
  if (feature.unlimited) {
    return '∞';
  }
  if (
    feature.includedUsage >= 1_000_000 &&
    feature.includedUsage % 1_000_000 === 0
  ) {
    return `${feature.includedUsage / 1_000_000}M`;
  }
  return new Intl.NumberFormat('en-US').format(feature.includedUsage);
}

function formatLimit(value?: number, fallback = '1') {
  if (value === undefined || value < 0) {
    return fallback;
  }
  return new Intl.NumberFormat('en-US').format(value);
}

function paygLimits(plan?: SubscriptionPlan) {
  const included = (key: keyof typeof PAYG_FEATURES) =>
    formatIncluded(
      findFeature(plan, PAYG_FEATURES[key].id),
      PAYG_FEATURES[key].fallback,
    );

  return {
    tenants: included('tenants'),
    users: included('users'),
    retentionDays: included('retentionDays'),
    concurrent: included('concurrent'),
    scheduled: included('scheduled'),
    crons: included('crons'),
    webhooks: included('webhooks'),
    taskRuns: included('taskRuns'),
    events: included('events'),
  };
}

function salesUrl(tenantName?: string, tenantId?: string) {
  if (!tenantName && !tenantId) {
    return OFFICE_HOURS_URL;
  }
  return `https://cal.com/team/hatchet/website-demo?notes=${encodeURIComponent(
    `Custom pricing request for tenant '${tenantName || 'Unknown'}' (${tenantId ?? ''})`,
  )}`;
}

function gateCopy(
  gate: UpgradeGate,
  payg: ReturnType<typeof paygLimits>,
  current: {
    tenants: string;
    users: string;
    retention: string;
  },
) {
  const pitch = {
    tenants: {
      title: 'Take Hatchet to production',
      context: PAYG_SUBHEAD,
    },
    users: {
      title: 'Bring your whole team',
      context: PAYG_SUBHEAD,
    },
    retention: {
      title: 'Debug with a week of history',
      context: `Switch to Pay as you Go to get ${payg.retentionDays}-day retention (up from ${current.retention} on the free tier).`,
    },
    usage: {
      title: 'Take Hatchet to production',
      context: PAYG_SUBHEAD,
    },
  }[gate];

  const infraLimits = `${payg.concurrent} worker slots, ${payg.scheduled} scheduled runs, ${payg.crons} crons, and ${payg.webhooks} webhook endpoints.`;
  const cards: Benefit[] = [
    {
      id: 'tenants',
      title: 'Isolated environments',
      description:
        gate === 'retention'
          ? `${payg.tenants} tenants for dev, staging, and prod (up from ${current.tenants} on the free tier); reproduce an issue in staging without touching prod.`
          : gate === 'users'
            ? `Separate dev, staging, and prod across ${payg.tenants} tenants (up from ${current.tenants} on free), so your team can build and test without stepping on toes.`
            : `Separate dev, staging, and prod across ${payg.tenants} tenants (up from ${current.tenants} on free), each with its own workers, keys, and limits.`,
      icon: BuildingOffice2Icon,
    },
    {
      id: 'retention',
      title: `${payg.retentionDays}-day retention`,
      description:
        gate === 'retention'
          ? 'A full week of runs, events, and logs to debug against, across every tenant and workflow.'
          : `A full week of runs, events, and logs to debug against (up from ${current.retention} on free).`,
      icon: ClockIcon,
    },
    {
      id: 'users',
      title: 'Built for teams shipping to production',
      description:
        gate === 'users'
          ? `${payg.users} teammates (up from ${current.users} on free), plus ${infraLimits}`
          : `${payg.users} teammates, ${infraLimits}`,
      icon: BoltIcon,
    },
    {
      id: 'usage',
      title: 'Same free volume, no monthly fee',
      description: `Your ${payg.taskRuns} task runs and ${payg.events} events stay included. You only pay for usage beyond that.`,
      icon: CurrencyDollarIcon,
    },
  ];

  return {
    pitch,
    cards: [
      cards.find((card) => card.id === gate) ?? cards[0],
      ...cards.filter((card) => card.id !== gate),
    ],
  };
}

export function UpgradeGateContent({
  gate,
  organizationId,
  onDismiss,
  retentionPeriod,
}: UpgradeGateProps) {
  const { canBill, isControlPlaneEnabled } = useControlPlane();
  const { tenant } = useTenantDetails();
  const { entitlements } = useOrganizationEntitlements(organizationId);
  const plansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: isControlPlaneEnabled && canBill,
  });
  const payg = payAsYouGoPlan(plansQuery.data?.plans);
  const upgrade = useSubscriptionUpgrade(organizationId);
  const { pitch, cards } = gateCopy(gate, paygLimits(payg), {
    tenants: formatLimit(entitlements?.tenants?.limit),
    users: formatLimit(entitlements?.users?.limit, '3'),
    retention: retentionPeriod
      ? formatRetentionPeriod(retentionPeriod)
      : '1 day',
  });

  return (
    <div className="flex flex-col gap-6">
      <div className="space-y-2 text-left">
        <h2 className="text-2xl font-semibold tracking-tight text-foreground">
          {pitch.title}
        </h2>
        <p className="text-sm text-muted-foreground">{pitch.context}</p>
      </div>

      <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
        {cards.map((card, index) => {
          const Icon = card.icon;
          return (
            <div
              key={card.id}
              className={cn(
                'rounded-lg border border-border/60 bg-muted/20 p-4 text-left',
                index === 0 && 'border-primary/40 bg-primary/5',
              )}
            >
              <Icon className="mb-3 h-5 w-5 text-foreground" />
              <p className="text-sm font-medium text-foreground">
                {card.title}
              </p>
              <p className="mt-1 text-sm text-muted-foreground">
                {card.description}
              </p>
            </div>
          );
        })}
      </div>

      {upgrade.isError ? (
        <Alert variant="destructive">
          <AlertTitle>Plan change failed</AlertTitle>
          <AlertDescription>
            {getPlanChangeErrorMessage(upgrade.error)}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="flex flex-col gap-3">
        <Button
          size="lg"
          className="w-full"
          disabled={
            !isControlPlaneEnabled || !canBill || !payg || upgrade.isPending
          }
          onClick={() => payg && upgrade.mutate(payg.planCode)}
        >
          {upgrade.isPending ? (
            <Spinner />
          ) : (
            'Upgrade to Pay as you Go – starts at $0/month'
          )}
        </Button>
        <a
          href={PRICING_URL}
          target="_blank"
          rel="noreferrer"
          className="text-center text-sm text-muted-foreground underline underline-offset-4 transition-colors hover:text-foreground"
        >
          More Pricing Details
        </a>
        <p className="text-center text-sm text-muted-foreground">
          Complex requirements? Volume discounts?{' '}
          <a
            href={salesUrl(tenant?.name, tenant?.metadata?.id)}
            target="_blank"
            rel="noreferrer"
            className="underline underline-offset-4 hover:text-foreground"
          >
            Let's chat
          </a>
        </p>
        {onDismiss ? (
          <button
            type="button"
            onClick={onDismiss}
            className="w-full py-1 text-center text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {DISMISS_LABEL}
          </button>
        ) : null}
      </div>
    </div>
  );
}

export function UpgradeGateDialog({
  open,
  onDismiss,
  ...contentProps
}: UpgradeGateProps & { open: boolean; onDismiss: () => void }) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onDismiss()}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogTitle className="sr-only">Upgrade to Pay as you Go</DialogTitle>
        <DialogDescription className="sr-only">
          Upgrade your plan to unlock more tenants, members, and retention.
        </DialogDescription>
        <UpgradeGateContent {...contentProps} onDismiss={onDismiss} />
      </DialogContent>
    </Dialog>
  );
}
