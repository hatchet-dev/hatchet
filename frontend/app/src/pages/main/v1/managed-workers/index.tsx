import { Separator } from '@/components/ui/separator';
import { Link } from 'react-router-dom';
import { ManagedWorkersTable } from './components/managed-workers-table';
import { Button } from '@/components/ui/button';
import { useTenant } from '@/lib/atoms';
import { Alert, AlertTitle, AlertDescription } from '@/components/ui/alert';
import { ExclamationTriangleIcon } from '@radix-ui/react-icons';
import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { useEffect, useState } from 'react';
import { managedCompute } from '@/lib/can/features/managed-compute';
import { RejectReason } from '@/lib/can/shared/permission.base';

export default function ManagedWorkers() {
  const { tenant, billing, can } = useTenant();

  const [portalLoading, setPortalLoading] = useState(false);

  // stop polling billing if there are payment methods
  useEffect(() => {
    if (billing?.hasPaymentMethods) {
      billing?.setPollBilling(false);
    }
  }, [billing, billing?.hasPaymentMethods]);

  const [canCreateManagedWorker, rejectReason] = can(managedCompute.create);

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

  if (rejectReason == RejectReason.BILLING_REQUIRED) {
    return (
      <div className="flex-grow h-full w-full">
        <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
          <Alert variant="warn" className="mb-4">
            <ExclamationTriangleIcon className="h-4 w-4" />
            <AlertTitle className="font-semibold">
              No Payment Method.
            </AlertTitle>
            <AlertDescription>
              A payment method is required to upgrade your subscription, please{' '}
              <a onClick={manageClicked} className="underline pointer" href="#">
                add one
              </a>{' '}
              first.
            </AlertDescription>
          </Alert>
        </div>
      </div>
    );
  }

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h2 className="text-2xl font-bold leading-tight text-foreground">
            Managed Compute
          </h2>
          {canCreateManagedWorker && (
            <Link to="/managed-workers/create">
              <Button>Deploy Workers</Button>
            </Link>
          )}
        </div>
        <Separator className="my-4" />
        <ManagedWorkersTable />
      </div>
    </div>
  );
}
