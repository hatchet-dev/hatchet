import { payAsYouGoPlan } from '@/components/v1/cloud/billing/subscription-plan-code';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import useControlPlane from '@/hooks/use-control-plane';
import {
  getPlanChangeErrorMessage,
  useSubscriptionUpgrade,
} from '@/hooks/use-subscription-upgrade';
import { useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';

export function FreePlanBanner() {
  const { billing, organizationId } = useTenantDetails();
  const { canBill, isControlPlaneEnabled } = useControlPlane();

  // Hide until billing has loaded, so paid plans never flash this banner.
  // A missing billing state (self-hosted, or a failed read) stays hidden too.
  const onFreePlan =
    !!organizationId &&
    !!billing?.state &&
    !billing.isLoading &&
    billing.plan === 'free';

  const plansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: onFreePlan && isControlPlaneEnabled && canBill,
  });
  const payg = payAsYouGoPlan(plansQuery.data?.plans);
  const upgrade = useSubscriptionUpgrade(organizationId);

  if (!onFreePlan || !organizationId) {
    return null;
  }

  return (
    <div className="flex flex-wrap items-center justify-between gap-4 rounded-lg border border-brand/40 bg-brand/5 p-4">
      <div className="space-y-1">
        <p className="text-sm font-medium">Remove free plan limits</p>
        <p className="text-sm text-muted-foreground">
          Add a credit card to lift the free plan limits. You pay nothing until
          you go past the included usage.
        </p>
        {upgrade.isError ? (
          <p className="text-sm text-destructive">
            {getPlanChangeErrorMessage(upgrade.error)}
          </p>
        ) : plansQuery.isError ? (
          <p className="text-sm text-destructive">
            We could not load plans. Refresh and try again.
          </p>
        ) : null}
      </div>
      <div className="flex flex-wrap items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          className="border-brand text-brand hover:bg-brand/10 hover:text-brand"
          disabled={!payg || upgrade.isPending}
          onClick={() => payg && upgrade.mutate(payg.planCode)}
        >
          {upgrade.isPending ? <Spinner /> : 'Add a credit card'}
        </Button>
        <Button
          variant="ghost"
          size="sm"
          className="text-muted-foreground"
          asChild
        >
          <Link
            to={appRoutes.organizationBillingRoute.to}
            params={{ organization: organizationId }}
          >
            More details
          </Link>
        </Button>
      </div>
    </div>
  );
}
