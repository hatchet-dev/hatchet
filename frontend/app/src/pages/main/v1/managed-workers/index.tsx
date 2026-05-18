import { BillingRequired } from './components/billing-required';
import { ManagedWorkersTable } from './components/managed-workers-table';
import { MonthlyUsageCard } from './components/monthly-usage-card';
import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { controlPlaneApi } from '@/lib/api/api';
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
      const link = await controlPlaneApi.request<{ url?: string }>({
        path: `/api/v1/control-plane/billing/tenants/${tenantId}/billing-portal-link`,
        method: 'GET',
        secure: true,
        format: 'json',
      });
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
      <Dialog open={showUpgradeModal} onOpenChange={setShowUpgradeModal}>
        <DialogContent className="max-w-md">
          <DialogHeader>
            <DialogTitle>Plan Upgrade Required</DialogTitle>
            <DialogDescription>
              You've reached the maximum number of services ({workerPoolCount}/
              {getWorkerPoolLimit()}) allowed on your current plan. Upgrade to
              create more services.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
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
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}
