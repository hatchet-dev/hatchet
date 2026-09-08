import { payAsYouGoPlan } from './subscription-plan-code';
import { formatPlanTier, useCurrentPlanName } from './upgrade-required';
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
import {
  TIME_WINDOW_LABELS,
  formatRetentionPeriod,
  formatShortDate,
} from '@/lib/utils/retention';
import {
  BoltIcon,
  BuildingOffice2Icon,
  ClockIcon,
  CurrencyDollarIcon,
} from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';

export type UpgradeGate = 'tenants' | 'users' | 'retention' | 'usage';

const TEAM_HEADLINE = 'Scale with your whole team';

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
    events: findFeature(plan, 'events'),
  };
}

function eventsDescription(events?: SubscriptionPlanFeature) {
  if (events?.display?.secondaryText) {
    return `No monthly fee. ${events.display.primaryText} included, ${events.display.secondaryText}.`;
  }
  return `No monthly fee. ${events?.display?.primaryText ?? '10M external events included'}, then usage-based pricing.`;
}

function comparedLimit(
  payg: string,
  current: string,
  noun: string,
  tier: string,
) {
  return `Pay as you Go includes ${payg} ${noun} — ${tier} includes ${current}.`;
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
    tier: string;
    tenants: string;
    users: string;
    retention: string;
    tried: string | null;
  },
) {
  const pitch = {
    tenants: {
      title: 'Separate dev, staging, and prod',
      context: comparedLimit(
        payg.tenants,
        current.tenants,
        'tenants',
        current.tier,
      ),
      dismiss: 'Not now',
    },
    users: {
      title: TEAM_HEADLINE,
      context: comparedLimit(
        payg.users,
        current.users,
        'members',
        current.tier,
      ),
      dismiss: 'Not now',
    },
    retention: {
      title: 'Debug with a week of history',
      context: [
        `Pay as you Go keeps ${payg.retentionDays} days of runs, events, and logs — ${current.tier} keeps ${current.retention}.`,
        current.tried,
      ]
        .filter(Boolean)
        .join(' '),
      dismiss: `Keep last ${current.retention}`,
    },
    usage: {
      title: TEAM_HEADLINE,
      context:
        'Pay as you Go raises included limits — no monthly fee, usage billed monthly.',
      dismiss: 'Not now',
    },
  }[gate];

  const cards: Benefit[] = [
    {
      id: 'tenants',
      title: 'Isolated environments',
      description: `${payg.tenants} tenants for dev, staging, and prod, each with its own workers, keys, and limits.`,
      icon: BuildingOffice2Icon,
    },
    {
      id: 'retention',
      title: `${payg.retentionDays}-day retention`,
      description: 'A week of runs, events, and logs to debug against.',
      icon: ClockIcon,
    },
    {
      id: 'users',
      title: TEAM_HEADLINE,
      description: `${payg.users} members, plus ${payg.concurrent} concurrent runs, ${payg.scheduled} scheduled runs, ${payg.crons} crons, and ${payg.webhooks} webhook endpoints.`,
      icon: BoltIcon,
    },
    {
      id: 'usage',
      title: 'Pay only for what you use',
      description: eventsDescription(payg.events),
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
  retentionAttempt,
  retentionPeriod,
}: UpgradeGateProps) {
  const { canBill, isControlPlaneEnabled } = useControlPlane();
  const { tenant } = useTenantDetails();
  const currentPlanName = useCurrentPlanName(organizationId);
  const { entitlements } = useOrganizationEntitlements(organizationId);
  const plansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: isControlPlaneEnabled && canBill,
  });
  const payg = payAsYouGoPlan(plansQuery.data?.plans);
  const upgrade = useSubscriptionUpgrade(organizationId);
  const { pitch, cards } = gateCopy(gate, paygLimits(payg), {
    tier: formatPlanTier(currentPlanName),
    tenants: formatLimit(entitlements?.tenants?.limit),
    users: formatLimit(entitlements?.users?.limit),
    retention: retentionPeriod
      ? formatRetentionPeriod(retentionPeriod)
      : '3 days',
    tried: retentionAttempt
      ? retentionAttempt.kind === 'preset'
        ? `You tried to view the last ${TIME_WINDOW_LABELS[retentionAttempt.window]}.`
        : `You tried to look back to ${formatShortDate(retentionAttempt.date)}.`
      : null,
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
          {upgrade.isPending ? <Spinner /> : 'Upgrade to Pay as you Go'}
        </Button>
        <a
          href={PRICING_URL}
          target="_blank"
          rel="noreferrer"
          className="text-center text-sm text-muted-foreground underline underline-offset-4 transition-colors hover:text-foreground"
        >
          More Details
        </a>
        <p className="text-center text-sm text-muted-foreground">
          Complex requirements? Volume discounts?{' '}
          <a
            href={salesUrl(tenant?.name, tenant?.metadata?.id)}
            target="_blank"
            rel="noreferrer"
            className="underline underline-offset-4 hover:text-foreground"
          >
            Talk to sales
          </a>
        </p>
        {onDismiss ? (
          <button
            type="button"
            onClick={onDismiss}
            className="w-full py-1 text-center text-sm text-muted-foreground transition-colors hover:text-foreground"
          >
            {pitch.dismiss}
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
