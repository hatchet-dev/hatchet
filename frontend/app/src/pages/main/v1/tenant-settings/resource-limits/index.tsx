import {
  limitDurationMap,
  limitedResources,
  LimitIndicator,
} from './components/resource-limit-columns';
import { PaymentMethods, Subscription } from '@/components/v1/cloud/billing';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import {
  Table,
  TableBody,
  TableCell,
  TableHeader,
  TableRow,
} from '@/components/v1/ui/table';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import useCloud from '@/pages/auth/hooks/use-cloud';
import { useQuery } from '@tanstack/react-query';

export default function ResourceLimits() {
  const { tenantId } = useCurrentTenantId();

  const { cloud, isCloudEnabled } = useCloud();

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(tenantId),
  });

  const billingState = useQuery({
    ...queries.cloud.billing(tenantId),
    enabled: isCloudEnabled && !!cloud?.canBill,
  });

  const billingEnabled = isCloudEnabled && cloud?.canBill;

  const hasPaymentMethods =
    (billingState.data?.paymentMethods?.length || 0) > 0;

  if (resourcePolicyQuery.isLoading || billingState.isLoading) {
    return (
      <div className="h-full w-full flex-grow px-4 sm:px-6 lg:px-8">
        <Spinner />
      </div>
    );
  }

  return (
    <div className="h-full w-full flex-grow">
      {billingEnabled && (
        <>
          <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
            <div className="flex flex-row items-center justify-between">
              <h2 className="text-2xl font-semibold leading-tight text-foreground">
                Billing and Limits
              </h2>
            </div>
          </div>
          <Separator className="my-4" />
          <PaymentMethods
            hasMethods={hasPaymentMethods}
            methods={billingState.data?.paymentMethods}
          />
          <Separator className="my-4" />
          <Subscription
            hasPaymentMethods={hasPaymentMethods}
            active={billingState.data?.subscription}
            plans={billingState.data?.plans}
            coupons={billingState.data?.coupons}
          />
          <Separator className="my-4" />
        </>
      )}

      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="flex flex-row items-center justify-between">
          <h3 className="text-xl font-semibold leading-tight text-foreground">
            Resource Limits
          </h3>
        </div>
        <p className="my-4 text-gray-700 dark:text-gray-300">
          Resource limits are used to control the usage of resources within a
          tenant. When a limit is reached, the system will take action based on
          the limit type. Please upgrade your plan, or{' '}
          <a href="https://hatchet.run/office-hours" className="underline">
            contact us
          </a>{' '}
          if you need to adjust your limits.
        </p>

        <Table className="border">
          <TableHeader>
            {[
              'Resource',
              'Current Value',
              'Limit Value',
              'Alarm Value',
              'Meter Window',
              'Last Refill',
            ].map((column) => (
              <TableCell key={column}>{column}</TableCell>
            ))}
          </TableHeader>
          <TableBody>
            {resourcePolicyQuery.data?.limits.map((limit) => (
              <TableRow key={limit.metadata.id}>
                <TableCell>
                  <div className="flex flex-row items-center gap-4">
                    <LimitIndicator
                      value={limit.value}
                      alarmValue={limit.alarmValue}
                      limitValue={limit.limitValue}
                    />
                    {limitedResources[limit.resource]}
                  </div>
                </TableCell>
                <TableCell>{limit.value}</TableCell>
                <TableCell>{limit.limitValue}</TableCell>
                <TableCell>{limit.alarmValue || 'N/A'}</TableCell>
                <TableCell>
                  {(limit.window || '-') in limitDurationMap
                    ? limitDurationMap[limit.window || '-']
                    : limit.window}
                </TableCell>
                <TableCell>
                  {!limit.window
                    ? 'N/A'
                    : limit.lastRefill && (
                        <RelativeDate date={limit.lastRefill} />
                      )}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
