import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { SimpleTable } from '@/components/v1/molecules/simple-table/simple-table';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Separator } from '@/components/v1/ui/separator';
import { Switch } from '@/components/v1/ui/switch';
import { useBilling } from '@/hooks/use-billing';
import { TenantResource, TenantResourceLimit } from '@/lib/api';
import {
  Organization,
  SubscriptionPlan,
} from '@/lib/api/generated/cloud/data-contracts';
import { cn } from '@/lib/utils';
import { useMemo, useState } from 'react';

const limitedResources: Record<TenantResource, string> = {
  [TenantResource.WORKER]: 'Total Workers',
  [TenantResource.WORKER_SLOT]: 'Concurrency Slots',
  [TenantResource.EVENT]: 'Events',
  [TenantResource.TASK_RUN]: 'Task Runs',
  [TenantResource.CRON]: 'Cron Triggers',
  [TenantResource.SCHEDULE]: 'Schedule Triggers',
  [TenantResource.INCOMING_WEBHOOK]: 'Incoming Webhooks',
};

const indicatorVariants = {
  ok: 'border-transparent rounded-full bg-green-500',
  alarm: 'border-transparent rounded-full bg-yellow-500',
  exhausted: 'border-transparent rounded-full bg-red-500',
};

function LimitIndicator({
  value,
  alarmValue,
  limitValue,
}: {
  value: number;
  alarmValue?: number;
  limitValue: number;
}) {
  let variant = indicatorVariants.ok;

  if (alarmValue && value >= alarmValue) {
    variant = indicatorVariants.alarm;
  }

  if (value >= limitValue) {
    variant = indicatorVariants.exhausted;
  }

  return <div className={cn(variant, 'h-[6px] w-[6px] rounded-full')} />;
}

const limitDurationMap: Record<string, string> = {
  '24h0m0s': 'Daily',
  '168h0m0s': 'Weekly',
  '720h0m0s': 'Monthly',
};

interface BillingTabProps {
  organization: Organization;
}

export function BillingTab({ organization }: BillingTabProps) {
  const [isChangeConfirmOpen, setChangeConfirmOpen] = useState<
    SubscriptionPlan | undefined
  >(undefined);

  const {
    isLoading,
    isError,
    changingPlanCode,
    portalLoading,
    coupons,
    currentPlanDetails,
    formattedEndDate,
    isDedicatedPlan,
    upcoming,
    upcomingPlanName,
    upcomingStartDate,
    availablePlans,
    showAnnual,
    setShowAnnual,
    isUpgrade,
    enterpriseContactUrl,
    openBillingPortal,
    changePlan,
    formatPrice,
    activeTenants,
    detailedTenants,
    selectedTenantId,
    setSelectedTenantId,
    resourcePolicyQuery,
  } = useBilling({ organization });

  const resourceLimitColumns = useMemo(
    () => [
      {
        columnLabel: 'Resource',
        cellRenderer: (limit: TenantResourceLimit) => (
          <div className="flex flex-row items-center gap-4">
            <LimitIndicator
              value={limit.value}
              alarmValue={limit.alarmValue}
              limitValue={limit.limitValue}
            />
            {limitedResources[limit.resource]}
          </div>
        ),
      },
      {
        columnLabel: 'Current Value',
        cellRenderer: (limit: TenantResourceLimit) => limit.value,
      },
      {
        columnLabel: 'Limit Value',
        cellRenderer: (limit: TenantResourceLimit) => limit.limitValue,
      },
      {
        columnLabel: 'Alarm Value',
        cellRenderer: (limit: TenantResourceLimit) => limit.alarmValue || 'N/A',
      },
      {
        columnLabel: 'Meter Window',
        cellRenderer: (limit: TenantResourceLimit) =>
          (limit.window || '-') in limitDurationMap
            ? limitDurationMap[limit.window || '-']
            : limit.window,
      },
      {
        columnLabel: 'Last Refill',
        cellRenderer: (limit: TenantResourceLimit) =>
          !limit.window
            ? 'N/A'
            : limit.lastRefill && <RelativeDate date={limit.lastRefill} />,
      },
    ],
    [],
  );

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Spinner />
      </div>
    );
  }

  if (isError) {
    return (
      <div className="py-8 text-center text-muted-foreground">
        Unable to load billing information.
      </div>
    );
  }

  return (
    <>
      <ConfirmDialog
        isOpen={!!isChangeConfirmOpen}
        title={'Confirm Change Plan'}
        submitVariant="default"
        description={
          <>
            Are you sure you'd like to change to {isChangeConfirmOpen?.name}{' '}
            plan?
            <br />
            <br />
            Upgrades will be prorated and downgrades will take effect at the end
            of the billing period.
          </>
        }
        submitLabel={'Change Plan'}
        onSubmit={async () => {
          await changePlan(isChangeConfirmOpen!);
          setChangeConfirmOpen(undefined);
        }}
        onCancel={() => setChangeConfirmOpen(undefined)}
        isLoading={!!changingPlanCode}
      />
      <div className="space-y-6">
        {isDedicatedPlan ? (
          <div className="flex flex-row items-center justify-between">
            <p className="text-xl font-semibold leading-tight text-foreground">
              You are on the Dedicated plan
            </p>
            <Button
              onClick={openBillingPortal}
              variant="outline"
              disabled={portalLoading}
            >
              {portalLoading ? <Spinner /> : 'Visit Billing Portal'}
            </Button>
          </div>
        ) : (
          <>
            <div className="flex flex-row items-center justify-between">
              <h3 className="flex flex-row gap-2 text-xl font-semibold leading-tight text-foreground">
                Subscription
                {coupons?.map((coupon, i) => (
                  <Badge key={`c${i}`} variant="successful">
                    {coupon.name} coupon applied
                  </Badge>
                ))}
              </h3>

              <Button
                onClick={openBillingPortal}
                variant="outline"
                disabled={portalLoading}
              >
                {portalLoading ? <Spinner /> : 'Visit Billing Portal'}
              </Button>
            </div>

            {currentPlanDetails && (
              <div>
                <h4 className="text-sm font-medium text-muted-foreground mb-3 uppercase tracking-wide">
                  Current Subscription
                </h4>
                <Card className="border-2 border-primary/20 bg-card">
                  <CardHeader className="pb-4">
                    <div className="flex items-start justify-between">
                      <div className="flex-1">
                        <CardTitle className="text-2xl mb-1">
                          {currentPlanDetails.name}
                        </CardTitle>
                        <div className="text-3xl font-bold mb-2">
                          {formatPrice(
                            currentPlanDetails.amountCents,
                            currentPlanDetails.period,
                          )}{' '}
                          <span className="text-base font-normal text-muted-foreground">
                            per month
                          </span>
                        </div>
                        {formattedEndDate && (
                          <p className="text-sm text-muted-foreground flex items-center gap-2">
                            Your service will end on {formattedEndDate}.
                          </p>
                        )}
                      </div>
                    </div>
                  </CardHeader>
                </Card>
              </div>
            )}

            {upcoming && upcoming.plan && (
              <Card className="border-2 border-orange-500/30 bg-orange-50 dark:bg-orange-950/20">
                <CardHeader className="pb-4">
                  <div className="flex items-start justify-between">
                    <div className="flex-1">
                      <div className="flex items-center gap-2 mb-2">
                        <Badge variant="inProgress">Scheduled Change</Badge>
                      </div>
                      <CardTitle className="text-lg mb-1">
                        Switching to {upcomingPlanName}
                      </CardTitle>
                      <p className="text-sm text-muted-foreground">
                        This change will take effect on {upcomingStartDate}
                      </p>
                    </div>
                  </div>
                </CardHeader>
              </Card>
            )}

            <div className="flex flex-row justify-between items-center">
              <p className="text-gray-700 dark:text-gray-300">
                For plan details, please visit{' '}
                <a
                  href="https://hatchet.run/pricing"
                  className="underline"
                  target="_blank"
                  rel="noreferrer"
                >
                  our pricing page
                </a>{' '}
                or{' '}
                <a
                  href="https://hatchet.run/office-hours"
                  className="underline"
                >
                  contact us
                </a>{' '}
                if you have custom requirements.
              </p>

              <div className="flex gap-2 items-center">
                <Switch
                  id="sa"
                  checked={showAnnual}
                  onClick={() => {
                    setShowAnnual((checkedState) => !checkedState);
                  }}
                />
                <Label htmlFor="sa" className="text-sm whitespace-nowrap">
                  Annual Billing{' '}
                  <Badge variant="inProgress" className="ml-2">
                    Save up to 20%
                  </Badge>
                </Label>
              </div>
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
              {availablePlans?.map((plan, i) => (
                <Card className="bg-muted/30 gap-4 flex-col flex" key={i}>
                  <CardHeader>
                    <CardTitle className="tracking-wide text-sm">
                      {plan.name}
                    </CardTitle>
                    <CardDescription className="py-4">
                      {formatPrice(plan.amountCents, plan.period)} per month
                      billed {plan.period}*
                    </CardDescription>
                    <CardDescription>
                      <Button
                        disabled={changingPlanCode === plan.planCode}
                        variant="default"
                        onClick={() => setChangeConfirmOpen(plan)}
                      >
                        {changingPlanCode === plan.planCode ? (
                          <Spinner />
                        ) : isUpgrade(plan) ? (
                          'Upgrade'
                        ) : (
                          'Downgrade'
                        )}
                      </Button>
                    </CardDescription>
                  </CardHeader>
                </Card>
              ))}
              <Card className="bg-muted/30 gap-4 flex-col flex">
                <CardHeader>
                  <CardTitle className="tracking-wide text-sm">
                    Enterprise
                  </CardTitle>
                  <CardDescription className="py-4">
                    Custom pricing
                  </CardDescription>
                  <CardDescription>
                    <Button
                      variant="default"
                      onClick={() =>
                        window.open(enterpriseContactUrl, '_blank')
                      }
                    >
                      Contact Us
                    </Button>
                  </CardDescription>
                </CardHeader>
              </Card>
            </div>
            <p className="text-sm text-gray-500">
              * subscription fee billed upfront{' '}
              {showAnnual ? 'yearly' : 'monthly'}, overages billed at the end of
              each month for usage in that month
            </p>
          </>
        )}

        <Separator className="my-4" />

        <div>
          <div className="flex flex-row items-center justify-between">
            <h3 className="text-xl font-semibold leading-tight text-foreground">
              Resource Limits
            </h3>
            {activeTenants.length > 1 && (
              <Select
                value={selectedTenantId}
                onValueChange={setSelectedTenantId}
              >
                <SelectTrigger className="w-[200px]">
                  <SelectValue placeholder="Select tenant" />
                </SelectTrigger>
                <SelectContent>
                  {activeTenants.map((tenant) => {
                    const detailedTenant = detailedTenants.find(
                      (t) => t?.metadata.id === tenant.id,
                    );
                    return (
                      <SelectItem key={tenant.id} value={tenant.id}>
                        {detailedTenant?.name || tenant.id}
                      </SelectItem>
                    );
                  })}
                </SelectContent>
              </Select>
            )}
          </div>
          <p className="my-4 text-gray-700 dark:text-gray-300">
            Resource limits are applied per tenant. When a limit is reached, the
            system will take action based on the limit type. Please upgrade your
            plan, or{' '}
            <a href="https://hatchet.run/office-hours" className="underline">
              contact us
            </a>{' '}
            if you need to adjust your limits.
          </p>

          {resourcePolicyQuery.isLoading ? (
            <div className="flex items-center justify-center py-8">
              <Spinner />
            </div>
          ) : (
            <SimpleTable
              columns={resourceLimitColumns}
              data={resourcePolicyQuery.data?.limits || []}
            />
          )}
        </div>
      </div>
    </>
  );
}
