import { usePylon } from '@/components/support-chat';
import {
  Subscription,
  SubscriptionHistory,
} from '@/components/v1/cloud/billing';
import { resolveSubscriptionPlanCode } from '@/components/v1/cloud/billing/subscription-plan-code';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Spinner } from '@/components/v1/ui/loading';
import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import type { TenantResourceLimit } from '@/lib/api';
import { getApiErrorStatus } from '@/lib/api/api';
import type { TenantResourceLimit as ControlPlaneTenantResourceLimit } from '@/lib/api/generated/control-plane/data-contracts';
import { useSearchParams } from '@/lib/router-helpers';
import { SettingsPageHeader } from '@/pages/main/v1/tenant-settings/components/settings-page-header';
import { TenantResourceLimitsTable } from '@/pages/main/v1/tenant-settings/resource-limits/components/tenant-resource-limits-table';
import { appRoutes } from '@/router';
import { useQuery } from '@tanstack/react-query';
import { useParams } from '@tanstack/react-router';
import { type ReactNode, useEffect, useMemo, useState } from 'react';

// While a plan change finalizes, poll billing state until the rendered active
// or upcoming plan reflects the expected plan code.
const SYNC_POLL_INTERVAL_MS = 2000;
const SYNC_TIMEOUT_MS = 90_000;
const SYNC_SUCCESS_DISMISS_MS = 6000;

type SyncState = 'idle' | 'syncing' | 'done' | 'timeout';

function BillingPageLayout({ children }: { children: ReactNode }) {
  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <SettingsPageHeader
          title="Billing"
          description="Manage your organization subscription, payment methods, and plan changes."
        />

        {children}
      </div>
    </div>
  );
}

function BillingMaintenanceCard() {
  const pylon = usePylon();

  return (
    <Card variant="light" className="mt-6">
      <CardHeader>
        <CardTitle>Billing maintenance</CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          We're making some upgrades to billing. Check back soon, or contact us
          if you need help.
        </p>
        {pylon.enabled && (
          <Button onClick={pylon.show} variant="outline">
            Contact us
          </Button>
        )}
      </CardContent>
    </Card>
  );
}

function BillingErrorCard({
  isRetrying,
  isUnauthorized,
  onRetry,
}: {
  isRetrying: boolean;
  isUnauthorized: boolean;
  onRetry: () => void;
}) {
  const pylon = usePylon();

  return (
    <Card variant="light" className="mt-6">
      <CardHeader>
        <CardTitle>
          {isUnauthorized ? 'Unauthorized' : 'Billing unavailable'}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        <p className="text-sm text-muted-foreground">
          {isUnauthorized
            ? 'You must be an organization owner to view billing and usage details.'
            : "We couldn't load this organization's billing and usage details. Please try again, or contact us if this keeps happening."}
        </p>
        <div className="flex flex-col gap-2 sm:flex-row">
          <Button onClick={onRetry} variant="outline" disabled={isRetrying}>
            {isRetrying ? <Spinner /> : 'Try again'}
          </Button>
          {pylon.enabled && (
            <Button onClick={pylon.show} variant="outline">
              Contact us
            </Button>
          )}
        </div>
      </CardContent>
    </Card>
  );
}

function toTenantResourceLimit(
  limit: ControlPlaneTenantResourceLimit,
): TenantResourceLimit {
  return {
    ...limit,
    resource: limit.resource as TenantResourceLimit['resource'],
  };
}

export default function OrganizationBillingPage() {
  const { controlPlaneMeta } = useControlPlane();

  if (controlPlaneMeta?.billingMaintenanceMode) {
    return (
      <BillingPageLayout>
        <BillingMaintenanceCard />
      </BillingPageLayout>
    );
  }

  return (
    <BillingPageLayout>
      <OrganizationBillingContent />
    </BillingPageLayout>
  );
}

function OrganizationBillingContent() {
  const { organization } = useParams({
    from: appRoutes.organizationsRoute.to,
  });
  const { cloud, isCloudEnabled } = useCloud();
  const [searchParams, setSearchParams] = useSearchParams();

  // Capture the sync hints once on mount; the URL is cleaned up immediately so
  // a reload does not restart stale polling. All authoritative state still
  // comes from the billing-state query below.
  const [expectedPlanCode] = useState<string | null>(() =>
    searchParams.get('billing_sync') === 'plan_change'
      ? searchParams.get('plan_code')
      : null,
  );
  const [syncState, setSyncState] = useState<SyncState>(() =>
    searchParams.get('billing_sync') === 'plan_change' &&
    searchParams.get('plan_code')
      ? 'syncing'
      : 'idle',
  );

  useEffect(() => {
    if (
      searchParams.get('billing_sync') === null &&
      searchParams.get('plan_code') === null
    ) {
      return;
    }

    setSearchParams(
      (prev) => {
        prev.delete('billing_sync');
        prev.delete('plan_code');
        return prev;
      },
      { replace: true },
    );
    // Run once on mount to strip the sync params from the URL.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  const isSyncing = syncState === 'syncing';

  const billingState = useQuery({
    ...queries.controlPlane.billing(organization),
    enabled: isCloudEnabled && !!cloud?.canBill,
    retry: (failureCount, error) =>
      getApiErrorStatus(error) !== 401 && failureCount < 3,
    refetchInterval: isSyncing ? SYNC_POLL_INTERVAL_MS : false,
  });

  const activePlanCode = resolveSubscriptionPlanCode(
    billingState.data?.currentSubscription,
    null,
  );
  const upcomingPlanCode = resolveSubscriptionPlanCode(
    billingState.data?.upcomingSubscription,
    null,
  );

  useEffect(() => {
    if (syncState !== 'syncing' || !expectedPlanCode) {
      return;
    }

    if (
      activePlanCode === expectedPlanCode ||
      upcomingPlanCode === expectedPlanCode
    ) {
      setSyncState('done');
    }
  }, [syncState, expectedPlanCode, activePlanCode, upcomingPlanCode]);

  useEffect(() => {
    if (syncState !== 'syncing') {
      return;
    }

    const timer = setTimeout(() => setSyncState('timeout'), SYNC_TIMEOUT_MS);
    return () => clearTimeout(timer);
  }, [syncState]);

  useEffect(() => {
    if (syncState !== 'done') {
      return;
    }

    const timer = setTimeout(
      () => setSyncState('idle'),
      SYNC_SUCCESS_DISMISS_MS,
    );
    return () => clearTimeout(timer);
  }, [syncState]);

  const expectedPlanName = useMemo(() => {
    if (!expectedPlanCode) {
      return null;
    }
    return (
      billingState.data?.plans?.find((p) => p.planCode === expectedPlanCode)
        ?.name ?? null
    );
  }, [billingState.data?.plans, expectedPlanCode]);

  const tenantResourceLimits = useQuery({
    ...queries.controlPlane.tenantResourceLimits(organization),
    enabled: isCloudEnabled,
  });

  const organizationTenants = tenantResourceLimits.data?.tenants ?? [];

  if (billingState.isError) {
    const status = getApiErrorStatus(billingState.error);
    const isUnauthorized = status === 401 || status === 403;

    return (
      <BillingErrorCard
        isRetrying={billingState.isFetching}
        isUnauthorized={isUnauthorized}
        onRetry={() => {
          void billingState.refetch();
        }}
      />
    );
  }

  return (
    <>
      {syncState === 'syncing' && (
        <Alert variant="info" className="mb-6">
          <AlertTitle className="flex items-center gap-2">
            <Spinner className="h-4 w-4" />
            Finalizing your plan change
          </AlertTitle>
          <AlertDescription>
            {expectedPlanName
              ? `We're activating the ${expectedPlanName} plan...`
              : "We're activating your new plan..."}
          </AlertDescription>
        </Alert>
      )}

      {syncState === 'done' && (
        <Alert variant="info" className="mb-6">
          <AlertTitle>Plan change complete</AlertTitle>
          <AlertDescription>
            {expectedPlanName
              ? `Your ${expectedPlanName} plan is now active.`
              : 'Your new plan is now active.'}
          </AlertDescription>
        </Alert>
      )}

      {syncState === 'timeout' && (
        <Alert variant="warn" className="mb-6">
          <AlertTitle>Your plan change is still processing</AlertTitle>
          <AlertDescription className="flex flex-col gap-3">
            <span>
              This is taking longer than expected. Your change may still be
              finalizing in the background.
            </span>
            <div>
              <Button
                size="sm"
                variant="outline"
                onClick={() => {
                  setSyncState('syncing');
                  void billingState.refetch();
                }}
              >
                Check again
              </Button>
            </div>
          </AlertDescription>
        </Alert>
      )}

      <Subscription
        active={billingState.data?.currentSubscription}
        upcoming={billingState.data?.upcomingSubscription}
        plans={billingState.data?.plans}
        coupons={billingState.data?.coupons}
      />

      {tenantResourceLimits.isLoading || organizationTenants.length > 0 ? (
        <div className="mt-12 space-y-8">
          <div>
            <h2 className="text-lg font-semibold text-foreground">
              Resource limits
            </h2>
            <p className="mt-1 text-sm text-muted-foreground">
              Usage limits applied to each tenant in this organization.
            </p>
          </div>

          {tenantResourceLimits.isLoading ? (
            <div className="py-6">
              <Spinner />
            </div>
          ) : (
            organizationTenants.map((tenant) => (
              <TenantResourceLimitsTable
                key={tenant.tenantId}
                tenantId={tenant.tenantId}
                tenantName={
                  tenant.tenantName || tenant.tenantSlug || tenant.tenantId
                }
                limits={tenant.limits.map(toTenantResourceLimit)}
              />
            ))
          )}
        </div>
      ) : null}

      <div className="mt-12 space-y-4">
        <div>
          <h2 className="text-lg font-semibold text-foreground">
            Plan history
          </h2>
          <p className="mt-1 text-sm text-muted-foreground">
            A record of subscription changes for this organization.
          </p>
        </div>

        <SubscriptionHistory
          history={billingState.data?.subscriptionHistory}
          plans={billingState.data?.plans}
        />
      </div>
    </>
  );
}
