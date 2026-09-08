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
  SubscriptionPlanFeatureGroup,
} from '@/lib/api/generated/control-plane/data-contracts';
import { OFFICE_HOURS_URL } from '@/lib/external-links';
import { cn } from '@/lib/utils';
import {
  TIME_WINDOW_LABELS,
  formatRetentionPeriod,
  formatShortDate,
} from '@/lib/utils/retention';
import {
  ArrowLeftIcon,
  BoltIcon,
  BuildingOffice2Icon,
  ClockIcon,
  CurrencyDollarIcon,
} from '@heroicons/react/24/outline';
import { CheckIcon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
import { useMemo, useState } from 'react';

export type UpgradeGate = 'tenants' | 'users' | 'retention';

type Benefit = {
  id: UpgradeGate | 'usage';
  title: string;
  description: string;
  icon: typeof BuildingOffice2Icon;
};

function findFeature(plan: SubscriptionPlan | undefined, featureId: string) {
  return plan?.featureGroups
    ?.flatMap((group) => group.features)
    .find((feature) => feature.featureId === featureId);
}

function formatIncluded(feature?: SubscriptionPlanFeature, fallback?: string) {
  if (!feature) {
    return fallback ?? '—';
  }
  if (feature.unlimited) {
    return '∞';
  }
  const value = feature.includedUsage;
  if (value >= 1_000_000 && value % 1_000_000 === 0) {
    return `${value / 1_000_000}M`;
  }
  return new Intl.NumberFormat('en-US').format(value);
}

function formatLimitCount(value?: number, fallback = '1') {
  if (value === undefined || value < 0) {
    return fallback;
  }
  return new Intl.NumberFormat('en-US').format(value);
}

function usageFeatures(plan?: SubscriptionPlan) {
  return (
    plan?.featureGroups?.find((group) => group.name === 'Usage')?.features ?? []
  ).filter((feature) => feature.included && feature.display?.primaryText);
}

function visibleGroups(
  plan?: SubscriptionPlan,
): SubscriptionPlanFeatureGroup[] {
  return (plan?.featureGroups ?? [])
    .map((group) => ({
      ...group,
      features: group.features.filter((feature) => feature.included),
    }))
    .filter((group) => group.features.length > 0);
}

export function UpgradeGateContent({
  gate,
  organizationId,
  onDismiss,
  retentionAttempt,
  retentionPeriod,
}: {
  gate: UpgradeGate;
  organizationId: string;
  onDismiss?: () => void;
  retentionAttempt?: RetentionAttempt | null;
  retentionPeriod?: string;
}) {
  const [screen, setScreen] = useState<'pitch' | 'plan'>('pitch');
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
  const canUpgrade = isControlPlaneEnabled && canBill && !!organizationId;

  const tenantCount = formatIncluded(findFeature(payg, 'tenants'), '10');
  const userCount = formatIncluded(findFeature(payg, 'users'), '50');
  const retentionDays = formatIncluded(
    findFeature(payg, 'data_retention_days'),
    '7',
  );
  const concurrent = formatIncluded(
    findFeature(payg, 'worker_slots_limit'),
    '500',
  );
  const scheduled = formatIncluded(findFeature(payg, 'scheduled_runs'), '1M');
  const crons = formatIncluded(findFeature(payg, 'crons'), '1,000');
  const webhooks = formatIncluded(findFeature(payg, 'webhooks'), '100');
  const events = findFeature(payg, 'events');

  const currentTier = formatPlanTier(currentPlanName);
  const currentTenantLimit = formatLimitCount(entitlements?.tenants?.limit);
  const currentUserLimit = formatLimitCount(entitlements?.users?.limit);
  const currentRetention = retentionPeriod
    ? formatRetentionPeriod(retentionPeriod)
    : '3 days';

  const tried = retentionAttempt
    ? retentionAttempt.kind === 'preset'
      ? `You tried to view the last ${TIME_WINDOW_LABELS[retentionAttempt.window]}.`
      : `You tried to look back to ${formatShortDate(retentionAttempt.date)}.`
    : null;

  const pitch = useMemo(() => {
    if (gate === 'tenants') {
      return {
        title: 'Separate dev, staging, and prod',
        context: `Pay as you Go includes ${tenantCount} tenants — ${currentTier} includes ${currentTenantLimit}.`,
        dismiss: 'Not now',
      };
    }
    if (gate === 'users') {
      return {
        title: 'Bring your whole team to Hatchet',
        context: `Pay as you Go includes ${userCount} members — ${currentTier} includes ${currentUserLimit}.`,
        dismiss: 'Not now',
      };
    }
    return {
      title: 'Debug with a week of history',
      context: [
        `Pay as you Go keeps ${retentionDays} days of runs, events, and logs — ${currentTier} keeps ${currentRetention}.`,
        tried,
      ]
        .filter(Boolean)
        .join(' '),
      dismiss: `Keep last ${currentRetention}`,
    };
  }, [
    currentRetention,
    currentTenantLimit,
    currentTier,
    currentUserLimit,
    gate,
    retentionDays,
    tenantCount,
    tried,
    userCount,
  ]);

  const benefits: Benefit[] = [
    {
      id: 'tenants',
      title: 'Isolated environments',
      description: `${tenantCount} tenants for dev, staging, and prod, each with its own workers, keys, and limits.`,
      icon: BuildingOffice2Icon,
    },
    {
      id: 'retention',
      title: `${retentionDays}-day retention`,
      description: `A week of runs, events, and logs to debug against.`,
      icon: ClockIcon,
    },
    {
      id: 'users',
      title: 'Invite the whole team',
      description: `${userCount} members, plus ${concurrent} concurrent runs, ${scheduled} scheduled runs, ${crons} crons, and ${webhooks} webhook endpoints.`,
      icon: BoltIcon,
    },
    {
      id: 'usage',
      title: 'Pay only for what you use',
      description: events?.display?.secondaryText
        ? `No monthly fee. ${events.display.primaryText} included, ${events.display.secondaryText}.`
        : `No monthly fee. ${events?.display?.primaryText ?? '10M external events included'}, then usage-based pricing.`,
      icon: CurrencyDollarIcon,
    },
  ];

  const orderedBenefits = [
    benefits.find((benefit) => benefit.id === gate) ?? benefits[0],
    ...benefits.filter((benefit) => benefit.id !== gate),
  ];

  const enterpriseContactUrl = tenant
    ? `https://cal.com/team/hatchet/website-demo?notes=${encodeURIComponent(
        `Custom pricing request for tenant '${tenant.name || 'Unknown'}' (${tenant.metadata?.id ?? ''})`,
      )}`
    : OFFICE_HOURS_URL;

  const handleUpgrade = () => {
    if (!payg) {
      return;
    }
    upgrade.mutate(payg.planCode);
  };

  return (
    <div className="flex flex-col gap-6">
      {screen === 'plan' ? (
        <button
          type="button"
          onClick={() => setScreen('pitch')}
          className="inline-flex items-center gap-1.5 self-start text-sm text-muted-foreground hover:text-foreground"
        >
          <ArrowLeftIcon className="h-4 w-4" />
          Back
        </button>
      ) : null}

      {screen === 'pitch' ? (
        <div className="space-y-2 text-left">
          <h2 className="text-2xl font-semibold tracking-tight text-foreground">
            {pitch.title}
          </h2>
          <p className="text-sm text-muted-foreground">{pitch.context}</p>
        </div>
      ) : (
        <div className="space-y-1 text-left">
          <h2 className="text-2xl font-semibold tracking-tight text-foreground">
            Pay as you Go details
          </h2>
          <p className="text-sm text-muted-foreground">
            No monthly fee. Usage billed monthly.
          </p>
        </div>
      )}

      {screen === 'pitch' ? (
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          {orderedBenefits.map((benefit, index) => {
            const Icon = benefit.icon;
            return (
              <div
                key={benefit.title}
                className={cn(
                  'rounded-lg border border-border/60 bg-muted/20 p-4 text-left',
                  index === 0 && 'border-primary/40 bg-primary/5',
                )}
              >
                <Icon className="mb-3 h-5 w-5 text-foreground" />
                <p className="text-sm font-medium text-foreground">
                  {benefit.title}
                </p>
                <p className="mt-1 text-sm text-muted-foreground">
                  {benefit.description}
                </p>
              </div>
            );
          })}
        </div>
      ) : (
        <div className="space-y-5">
          {usageFeatures(payg).length > 0 ? (
            <div className="divide-y divide-border/50 rounded-lg border border-border/60">
              {usageFeatures(payg).map((feature) => (
                <div
                  key={feature.featureId}
                  className="flex items-start justify-between gap-4 px-4 py-3"
                >
                  <div>
                    <p className="text-sm font-medium text-foreground">
                      {feature.display?.primaryText ?? feature.name}
                    </p>
                    {feature.display?.secondaryText ? (
                      <p className="mt-0.5 text-xs text-muted-foreground">
                        {feature.display.secondaryText}
                      </p>
                    ) : null}
                  </div>
                </div>
              ))}
            </div>
          ) : null}

          <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
            {visibleGroups(payg).map((group) => (
              <div key={group.name} className="space-y-2">
                <p className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
                  {group.name}
                </p>
                <ul className="space-y-1.5">
                  {group.features.map((feature) => (
                    <li
                      key={feature.featureId}
                      className="flex items-start gap-2 text-sm"
                    >
                      <CheckIcon className="mt-0.5 size-3.5 shrink-0 text-primary" />
                      <span className="text-foreground">
                        {feature.display?.primaryText ?? feature.name}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        </div>
      )}

      {upgrade.isError ? (
        <Alert variant="destructive">
          <AlertTitle>Plan change failed</AlertTitle>
          <AlertDescription>
            {getPlanChangeErrorMessage(upgrade.error)}
          </AlertDescription>
        </Alert>
      ) : null}

      <div className="flex flex-col gap-3">
        <div className="flex flex-col gap-2">
          <Button
            size="lg"
            className="w-full"
            disabled={!canUpgrade || !payg || upgrade.isPending}
            onClick={handleUpgrade}
          >
            {upgrade.isPending ? <Spinner /> : 'Upgrade to Pay as you Go'}
          </Button>
          {screen === 'pitch' ? (
            <div className="grid grid-cols-2 gap-2">
              <Button variant="outline" onClick={() => setScreen('plan')}>
                Details
              </Button>
              <Button variant="outline" asChild>
                <a
                  href={enterpriseContactUrl}
                  target="_blank"
                  rel="noreferrer"
                >
                  Talk to sales
                </a>
              </Button>
            </div>
          ) : (
            <Button variant="outline" className="w-full" asChild>
              <a href={enterpriseContactUrl} target="_blank" rel="noreferrer">
                Talk to sales
              </a>
            </Button>
          )}
        </div>
        <p className="text-center text-xs text-muted-foreground">
          No monthly fee. Cancel anytime.
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
  gate,
  organizationId,
  onDismiss,
  retentionAttempt,
  retentionPeriod,
}: {
  open: boolean;
  gate: UpgradeGate;
  organizationId: string;
  onDismiss: () => void;
  retentionAttempt?: RetentionAttempt | null;
  retentionPeriod?: string;
}) {
  return (
    <Dialog open={open} onOpenChange={(next) => !next && onDismiss()}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogTitle className="sr-only">Upgrade to Pay as you Go</DialogTitle>
        <DialogDescription className="sr-only">
          Upgrade your plan to unlock more tenants, members, and retention.
        </DialogDescription>
        <UpgradeGateContent
          gate={gate}
          organizationId={organizationId}
          onDismiss={onDismiss}
          retentionAttempt={retentionAttempt}
          retentionPeriod={retentionPeriod}
        />
      </DialogContent>
    </Dialog>
  );
}
