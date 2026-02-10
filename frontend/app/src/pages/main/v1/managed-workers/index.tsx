import { BillingRequired } from './components/billing-required';
import { ManagedWorkersTable } from './components/managed-workers-table';
import { MonthlyUsageCard } from './components/monthly-usage-card';
import { Button } from '@/components/v1/ui/button';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { cloudApi } from '@/lib/api/api';
import { queries } from '@/lib/api/queries';
import { managedCompute } from '@/lib/can/features/managed-compute';
import { RejectReason } from '@/lib/can/shared/permission.base';
import { useApiError } from '@/lib/hooks';
import { appRoutes } from '@/router';
import { PlusIcon, ArrowUpIcon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';
import { useEffect, useState } from 'react';

export default function ManagedWorkers() {
  const { tenant, billing, can } = useTenantDetails();
  const { tenantId } = useCurrentTenantId();

  const [portalLoading, setPortalLoading] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);

  const computeCostQuery = useQuery({
    ...queries.cloud.getComputeCost(tenantId),
  });

  const listManagedWorkersQuery = useQuery({
    ...queries.cloud.listManagedWorkers(tenantId),
  });

  // Check if the user can create more worker pools
  const workerPoolCount = listManagedWorkersQuery.data?.rows?.length || 0;
  const [canCreateMoreWorkerPools] = can(
    managedCompute.canCreateWorkerPool(workerPoolCount),
  );

  // stop polling billing if there are payment methods
  useEffect(() => {
    if (billing?.hasPaymentMethods) {
      billing?.setPollBilling(false);
    }
  }, [billing, billing?.hasPaymentMethods]);

  const [, rejectReason] = can(managedCompute.create());

  const { handleApiError } = useApiError({});

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      billing?.setPollBilling(true);
      if (!tenantId) {
        return;
      }
      const link = await cloudApi.billingPortalLinkGet(tenantId);
      window.open(link.data.url, '_blank');
    } catch (e) {
      handleApiError(e as any);
    } finally {
      setPortalLoading(false);
    }
  };

  // Only show BillingRequired if there are no managed workers AND billing is required
  const hasExistingWorkers =
    (listManagedWorkersQuery.data?.rows?.length || 0) > 0;

  // Show loader while billing data is loading
  if (billing?.isLoading) {
    return (
      <div className="flex items-center justify-center h-screen">
        <Spinner />
      </div>
    );
  }

  // Don't show billing required page while billing data is still loading
  if (rejectReason == RejectReason.BILLING_REQUIRED && !hasExistingWorkers) {
    return (
      <BillingRequired
        tenant={tenant}
        billing={billing}
        manageClicked={manageClicked}
        portalLoading={portalLoading}
      />
    );
  }

  // Get limit based on plan
  const getWorkerPoolLimit = () => {
    if (!billing?.plan) {
      return 0;
    }

    switch (billing.plan) {
      case 'free':
        return 1;
      case 'starter':
        return 2;
      case 'growth':
        return 5;
      default:
        // This covers 'enterprise' and any other plans
        return 5;
    }
  };

  // Handler for when a user tries to add a worker pool but has reached their limit
  const handleAddWorkerPool = () => {
    if (!canCreateMoreWorkerPools) {
      setShowUpgradeModal(true);
    }
  };

  const UpgradeModal = () => {
    if (!showUpgradeModal) {
      return null;
    }

    return (
      // TODO use correct modal component
      <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 dark:bg-black/70">
        <div className="w-full max-w-md rounded-lg border border-border bg-background p-6 shadow-lg">
          <h3 className="mb-4 text-lg font-medium text-foreground">
            Plan Upgrade Required
          </h3>
          <p className="mb-4 text-muted-foreground">
            You've reached the maximum number of services ({workerPoolCount}/
            {getWorkerPoolLimit()}) allowed on your current plan. Upgrade to
            create more services.
          </p>
          <div className="flex justify-end gap-3">
            <Button
              variant="outline"
              onClick={() => setShowUpgradeModal(false)}
            >
              Cancel
            </Button>
            <Link
              to={appRoutes.tenantSettingsBillingRoute.to}
              params={{ tenant: tenantId }}
            >
              <Button leftIcon={<ArrowUpIcon className="size-4" />}>
                Upgrade Plan
              </Button>
            </Link>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center justify-between">
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Managed Compute
          </h2>
          {canCreateMoreWorkerPools ? (
            <Link
              to={appRoutes.tenantManagedWorkersCreateRoute.to}
              params={{ tenant: tenantId }}
            >
              <Button leftIcon={<PlusIcon className="size-4" />}>
                Add Service
              </Button>
            </Link>
          ) : (
            <Button
              onClick={handleAddWorkerPool}
              leftIcon={<PlusIcon className="size-4" />}
            >
              Add Service ({workerPoolCount}/{getWorkerPoolLimit()})
            </Button>
          )}
        </div>
        <Separator className="my-4" />
        {listManagedWorkersQuery.isLoading ? (
          <div>Loading...</div>
        ) : (
          <ManagedWorkersTable />
        )}
        <div className="mb-6 mt-6">
          <MonthlyUsageCard
            computeCost={computeCostQuery.data}
            isLoading={computeCostQuery.isLoading}
          />
        </div>
      </div>
      <UpgradeModal />
    </div>
  );
}
