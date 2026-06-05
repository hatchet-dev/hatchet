import { resolveSubscriptionPlanCode } from './subscription-plan-code';
import { Button } from '@/components/v1/ui/button';
import useCloud from '@/hooks/use-cloud';
import { useOrganizationEntitlements } from '@/hooks/use-organization-entitlements';
import { queries } from '@/lib/api';
import { appRoutes } from '@/router';
import { ArrowUpCircleIcon } from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

export type UpgradeResource = 'users' | 'tenants';

type UpgradeRequiredCardProps = {
  resource: UpgradeResource;
  organizationId: string;
  onNavigate?: () => void;
};

const COPY: Record<
  UpgradeResource,
  { title: string; description: string; noun: string }
> = {
  users: {
    title: "You've reached your plan's member limit",
    description:
      'Your current plan does not allow inviting additional members. Upgrade your plan to add more people to your organization and tenants.',
    noun: 'members',
  },
  tenants: {
    title: "You've reached your plan's tenant limit",
    description:
      'Your current plan does not allow creating additional tenants. Upgrade your plan to create more tenants in your organization.',
    noun: 'tenants',
  },
};

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
  const { title, description, noun } = COPY[resource];
  const navigate = useNavigate();

  const { cloud, isCloudEnabled } = useCloud();
  const { entitlements } = useOrganizationEntitlements(organizationId);

  const billingState = useQuery({
    ...queries.controlPlane.billing(organizationId),
    enabled: isCloudEnabled && !!cloud?.canBill && !!organizationId,
  });

  const limit = entitlements?.[resource];

  const currentPlanCode = resolveSubscriptionPlanCode(
    billingState.data?.currentSubscription,
    null,
  );
  const currentPlanName =
    billingState.data?.plans?.find((p) => p.planCode === currentPlanCode)
      ?.name ??
    billingState.data?.currentSubscription?.plan ??
    null;

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
    <div className="mx-auto flex max-w-md flex-col items-center py-4 text-center">
      <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
        <ArrowUpCircleIcon className="h-8 w-8 text-primary" />
      </div>

      <h3 className="mb-2 text-xl font-semibold">{title}</h3>

      <p className="text-muted-foreground mb-6 text-sm">{description}</p>

      {showSummary && (
        <div className="mb-6 w-full rounded-lg border border-border bg-muted/30 p-4 text-sm">
          {currentPlanName && (
            <div className="flex items-center justify-between">
              <span className="text-muted-foreground">Current plan</span>
              <span className="font-medium text-foreground">
                {currentPlanName}
              </span>
            </div>
          )}
          {limit && !limit.unlimited && (
            <div className="mt-2 flex items-center justify-between">
              <span className="text-muted-foreground capitalize">{noun}</span>
              <span className="font-medium text-foreground">
                {limit.used} of {limit.limit} used
              </span>
            </div>
          )}
        </div>
      )}

      <Button size="lg" className="w-full" onClick={handleUpgrade}>
        View plans &amp; upgrade
      </Button>
    </div>
  );
}
