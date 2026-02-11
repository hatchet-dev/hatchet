import { Button } from '@/components/v1/ui/button';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { Tenant } from '@/lib/api';
import { queries } from '@/lib/api/queries';
import { BillingContext } from '@/lib/atoms';
import { appRoutes } from '@/router';
import {
  CalendarIcon,
  CpuChipIcon,
  CurrencyDollarIcon,
} from '@heroicons/react/24/outline';
import { useQuery } from '@tanstack/react-query';
import { Link } from '@tanstack/react-router';

interface BillingRequiredProps {
  tenant?: Tenant | undefined;
  billing?: BillingContext | undefined;
  manageClicked: () => Promise<void>;
  portalLoading: boolean;
}

export function BillingRequired({
  tenant,
  manageClicked,
  portalLoading,
}: BillingRequiredProps) {
  const { tenantId } = useCurrentTenantId();
  // Query for compute cost information to show available credits
  const computeCostQuery = useQuery({
    ...queries.cloud.getComputeCost(tenant?.metadata.id || ''),
    enabled: !!tenant?.metadata.id,
  });

  const hasCredits =
    computeCostQuery.data?.hasCreditsRemaining &&
    computeCostQuery.data?.creditsRemaining !== undefined;

  return (
    <div className="h-full w-full flex-grow">
      <div className="mx-auto px-4 py-8 sm:px-6 lg:px-8">
        <div className="rounded-lg border bg-card p-12 shadow-sm">
          <div className="mx-auto flex max-w-md flex-col items-center text-center">
            <div className="mb-6 flex h-16 w-16 items-center justify-center rounded-full bg-primary/10">
              <CpuChipIcon className="h-8 w-8 text-primary" />
            </div>

            <h3 className="mb-2 text-2xl font-semibold">
              Ready to supercharge your task runs?
            </h3>

            <p className="text-muted-foreground mb-6">
              Unlock Managed Compute by setting up billing. No commitment
              required - you only pay for what you use!
            </p>

            {/* Pricing Information */}
            <div className="mb-6 w-full rounded-lg border bg-muted/10 p-4">
              <div className="flex items-start">
                <div className="mr-3 flex h-10 w-10 items-center justify-center rounded-full bg-primary/10">
                  <CurrencyDollarIcon className="h-5 w-5 text-primary" />
                </div>
                <div className="flex-1 text-left">
                  <h4 className="font-medium">
                    Transparent Pay as You Go Pricing
                  </h4>
                  <div className="mt-2 grid gap-2 text-sm">
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">CPU:</span>
                      <span className="font-medium">$0.01/hour/CPU</span>
                    </div>
                    <div className="flex justify-between">
                      <span className="text-muted-foreground">Memory:</span>
                      <span className="font-medium">$0.01/hour/GB RAM</span>
                    </div>
                    {hasCredits &&
                      computeCostQuery.data?.creditsRemaining !== undefined && (
                        <div className="flex justify-between">
                          <span className="text-muted-foreground">
                            Monthly Free Credits:
                          </span>
                          <span className="font-medium text-green-500">
                            {new Intl.NumberFormat('en-US', {
                              style: 'currency',
                              currency: 'USD',
                            }).format(computeCostQuery.data.creditsRemaining)}
                          </span>
                        </div>
                      )}
                  </div>
                </div>
              </div>
            </div>

            <div className="mb-8 flex justify-center rounded-lg bg-muted/30 p-6">
              <div className="grid grid-cols-2 gap-x-8 gap-y-3 text-left text-sm">
                <div className="flex items-start">
                  <span className="mr-2 flex items-center text-primary">•</span>
                  <span>Auto-scaling workers based on slots</span>
                </div>
                <div className="flex items-start">
                  <span className="mr-2 flex items-center text-primary">•</span>
                  <span>Zero infrastructure headaches</span>
                </div>
                <div className="flex items-start">
                  <span className="mr-2 flex items-center text-primary">•</span>
                  <span>Multiple regions and machine types</span>
                </div>
                <div className="flex items-start">
                  <span className="mr-2 flex items-center text-primary">•</span>
                  <span>No cold starts</span>
                </div>
              </div>
            </div>

            <div className="flex w-full flex-col gap-4">
              <Button
                onClick={manageClicked}
                disabled={portalLoading}
                className="min-w-40 px-8 py-6 text-base"
                size="lg"
              >
                {portalLoading ? 'Loading...' : 'Set Up Billing →'}
              </Button>

              <div className="relative">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">
                    Not ready yet?
                  </span>
                </div>
              </div>

              <Link
                to={appRoutes.tenantManagedWorkersTemplateRoute.to}
                params={{ tenant: tenantId }}
                className="w-full"
              >
                <Button
                  variant="outline"
                  className="w-full min-w-40 px-8 py-6 text-base"
                  size="lg"
                >
                  Deploy a Demo Template for Free
                </Button>
              </Link>

              <div className="relative mt-4">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">
                    Have questions or requirements?
                  </span>
                </div>
              </div>

              <a
                href="https://hatchet.run/office-hours"
                target="_blank"
                rel="noopener noreferrer"
                className="w-full"
              >
                <Button
                  variant="ghost"
                  className="flex w-full min-w-40 items-center justify-center gap-2 px-8 py-6 text-base"
                  size="lg"
                >
                  <CalendarIcon className="h-5 w-5" />
                  Book a Call with Our Team
                </Button>
              </a>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
