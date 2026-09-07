import { resolveSubscriptionPlanCode } from './subscription-plan-code';
import { Button } from '@/components/v1/ui/button';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizationEntitlements } from '@/hooks/use-organization-entitlements';
import { queries } from '@/lib/api';
import { appRoutes } from '@/router';
import { ArrowUpCircleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { ReactNode } from 'react';
import { useNavigate } from '@tanstack/react-router';

export type UpgradeResource = 'users' | 'tenants';

type UpgradeRequiredCardProps = {
  resource: UpgradeResource;
  organizationId: string;
  onNavigate?: () => void;
};

const RESOURCE_COPY: Record<
  UpgradeResource,
  { noun: string; description: (tier: string) => string }
> = {
  users: {
    noun: 'member',
    description: (tier) =>
      `The ${tier} does not allow inviting additional members. Upgrade your plan to add more people to your organization and tenants.`,
  },
  tenants: {
    noun: 'tenant',
    description: (tier) =>
      `The ${tier} does not allow creating additional tenants. Upgrade your plan to create more tenants in your organization.`,
  },
};

function nonempty(value: string | null | undefined) {
  const trimmed = value?.trim();
  return trimmed ? trimmed : null;
}

/** "Free" → "free tier", "Free Plan" → "free plan" */
export function formatPlanTier(planName: string | null | undefined) {
  const lower = nonempty(planName)?.toLowerCase();
  if (!lower) {
    return 'free tier';
  }
  if (lower.endsWith('tier') || lower.endsWith('plan')) {
    return lower;
  }

  return `${lower} tier`;
}

export function useCurrentPlanName(organizationId?: string | null) {
  const { canBill, isControlPlaneEnabled } = useControlPlane();

  const billingState = useQuery({
    ...queries.controlPlane.billing(organizationId ?? ''),
    enabled: isControlPlaneEnabled && canBill && !!organizationId,
  });

  const currentPlanCode = nonempty(
    resolveSubscriptionPlanCode(
      billingState.data?.currentSubscription,
      null,
    ),
  );
  if (currentPlanCode) {
    return (
      billingState.data?.plans?.find((p) => p.planCode === currentPlanCode)
        ?.name ?? currentPlanCode
    );
  }

  const plan = nonempty(billingState.data?.currentSubscription?.plan);
  if (plan) {
    return (
      billingState.data?.plans?.find((p) => p.planCode === plan)?.name ?? plan
    );
  }

  // No subscription row (or an empty currentSubscription) is the free tier.
  return (
    billingState.data?.plans?.find((p) => p.planCode === 'free')?.name ?? 'Free'
  );
}

/**
 * Shared upgrade surface: icon, title, copy, optional summary, actions.
 * Used by invite/tenant limit cards and the retention modal.
 */
export function UpgradeRequiredLayout({
  title,
  description,
  summary,
  children,
}: {
  title: string;
  description: ReactNode;
  summary?: ReactNode;
  children?: ReactNode;
}) {
  return (
    <div className="mx-auto flex max-w-md flex-col items-center py-4 text-center">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
        <ArrowUpCircleIcon className="h-8 w-8 text-primary" />
      </div>

      <h3 className="mb-2 text-xl font-semibold">{title}</h3>

      <div className="text-muted-foreground mb-6 space-y-2 text-sm">
        {description}
      </div>

      {summary ? (
        <div className="mb-6 w-full rounded-lg border border-border bg-muted/30 p-4 text-left text-sm">
          {summary}
        </div>
      ) : null}

      {children}
    </div>
  );
}

/**
 * Upgrade surface shown in place of an invite or tenant-creation form when the
 * organization has reached its plan's entitlement limit. Renders plain content
 * so it can be embedded inside a dialog or a full page.
 */
export function UpgradeRequiredCard({
  resource,
  organizationId,
  onNavigate,
}: UpgradeRequiredCardProps) {
  const { noun, description } = RESOURCE_COPY[resource];
  const navigate = useNavigate();
  const { entitlements } = useOrganizationEntitlements(organizationId);
  const currentPlanName = useCurrentPlanName(organizationId);
  const tier = formatPlanTier(currentPlanName);
  const limit = entitlements?.[resource];

  const handleUpgrade = () => {
    // Dismiss the host modal (if any) before navigating so it doesn't linger
    // on top of the billing page after the route changes.
    onNavigate?.();
    void navigate({
      to: appRoutes.organizationBillingRoute.to,
      params: { organization: organizationId },
      hash: 'plan-selector',
    });
  };

  const showSummary = !!currentPlanName || (!!limit && !limit.unlimited);

  return (
    <UpgradeRequiredLayout
      title={`You've reached the ${tier}'s ${noun} limit`}
      description={<p>{description(tier)}</p>}
      summary={
        showSummary ? (
          <>
            {currentPlanName ? (
              <div className="flex items-center justify-between">
                <span className="text-muted-foreground">Current plan</span>
                <span className="font-medium text-foreground">
                  {currentPlanName}
                </span>
              </div>
            ) : null}
            {limit && !limit.unlimited ? (
              <div className="mt-2 flex items-center justify-between">
                <span className="text-muted-foreground capitalize">
                  {noun}s
                </span>
                <span className="font-medium text-foreground">
                  {limit.used} of {limit.limit} used
                </span>
              </div>
            ) : null}
          </>
        ) : undefined
      }
    >
      <Button size="lg" className="w-full" onClick={handleUpgrade}>
        View plans &amp; upgrade
      </Button>
    </UpgradeRequiredLayout>
  );
}
