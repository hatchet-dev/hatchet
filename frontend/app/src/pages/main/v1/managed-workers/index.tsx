import { Separator } from '@/components/ui/separator';
import { Link } from 'react-router-dom';
import { ManagedWorkersTable } from './components/managed-workers-table';
import { Button } from '@/components/ui/button';
import { useTenant } from '@/lib/atoms';
import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { useEffect, useState } from 'react';
import { managedCompute } from '@/lib/can/features/managed-compute';
import { RejectReason } from '@/lib/can/shared/permission.base';
import { BillingRequired } from './components/billing-required';
import { queries } from '@/lib/api/queries';
import { useQuery } from '@tanstack/react-query';
import { PlusIcon, ArrowUpIcon } from '@radix-ui/react-icons';
import { MonthlyUsageCard } from './components/monthly-usage-card';

export default function ManagedWorkers() {
  const { tenant, billing, can } = useTenant();

  const [portalLoading, setPortalLoading] = useState(false);
  const [showUpgradeModal, setShowUpgradeModal] = useState(false);

  const computeCostQuery = useQuery({
    ...queries.cloud.getComputeCost(tenant!.metadata.id),
  });

  const listManagedWorkersQuery = useQuery({
    ...queries.cloud.listManagedWorkers(tenant!.metadata.id),
  });

  // Check if the user can create more worker pools
  const workerPoolCount = listManagedWorkersQuery.data?.rows?.length || 0;
  const [canCreateMoreWorkerPools, createWorkerPoolsRejectReason] = can(
    managedCompute.canCreateWorkerPool(workerPoolCount),
  );

  // stop polling billing if there are payment methods
  useEffect(() => {
    if (billing?.hasPaymentMethods) {
      billing?.setPollBilling(false);
    }
  }, [billing, billing?.hasPaymentMethods]);

  const [canCreateManagedWorker, rejectReason] = can(managedCompute.create());

  const { handleApiError } = useApiError({});

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      billing?.setPollBilling(true);
      const link = await cloudApi.billingPortalLinkGet(tenant!.metadata.id);
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
      <div className="fixed inset-0 bg-black/50 dark:bg-black/70 flex items-center justify-center z-50">
        <div className="bg-background border border-border rounded-lg shadow-lg max-w-md w-full p-6">
          <h3 className="text-lg font-medium mb-4 text-foreground">
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
            <Link to="/v1/tenant-settings/billing-and-limits">
              <Button>
                <ArrowUpIcon className="h-4 w-4 mr-2" />
                Upgrade Plan
              </Button>
            </Link>
          </div>
        </div>
      </div>
    );
  };

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Managed Compute
          </h2>
          {canCreateMoreWorkerPools ? (
            <Link to="/v1/managed-workers/create">
              <Button>
                <PlusIcon className="w-4 h-4 mr-2" />
                Add Service
              </Button>
            </Link>
          ) : (
            <Button onClick={handleAddWorkerPool}>
              <PlusIcon className="w-4 h-4 mr-2" />
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
        <div className="mt-6 mb-6">
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
