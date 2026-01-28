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
import api, { queries, TenantResource, TenantResourceLimit } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import {
  Organization,
  SubscriptionPlan,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { cn } from '@/lib/utils';
import {
  useQuery,
  useQueries,
  useMutation,
  useQueryClient,
} from '@tanstack/react-query';
import { useEffect, useMemo, useState, useCallback } from 'react';

const ALLOWED_PLAN_CODES = [
  'free',
  'starter_monthly',
  'starter_yearly',
  'growth_monthly',
  'growth_yearly',
];

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
  orgId: string;
}

export function BillingTab({ organization, orgId }: BillingTabProps) {
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});

  const [loading, setLoading] = useState<string>();
  const [showAnnual, setShowAnnual] = useState<boolean>(false);
  const [isChangeConfirmOpen, setChangeConfirmOpen] = useState<
    SubscriptionPlan | undefined
  >(undefined);
  const [portalLoading, setPortalLoading] = useState(false);

  const billingQuery = useQuery({
    queryKey: ['organization-billing', orgId],
    queryFn: async () => {
      const result = await cloudApi.organizationBillingStateGet(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  const activeTenants = useMemo(
    () =>
      (organization.tenants || []).filter(
        (t) => t.status !== TenantStatusType.ARCHIVED,
      ),
    [organization.tenants],
  );

  const tenantQueries = useQueries({
    queries: activeTenants.map((tenant) => ({
      queryKey: ['tenant:get', tenant.id],
      queryFn: async () => {
        const result = await api.tenantGet(tenant.id);
        return result.data;
      },
      enabled: !!tenant.id,
    })),
  });

  const detailedTenants = useMemo(
    () =>
      tenantQueries.filter((query) => query.data).map((query) => query.data),
    [tenantQueries],
  );

  const [selectedTenantId, setSelectedTenantId] = useState<string | undefined>(
    activeTenants[0]?.id,
  );

  useEffect(() => {
    if (!selectedTenantId && activeTenants.length > 0) {
      setSelectedTenantId(activeTenants[0].id);
    }
  }, [activeTenants, selectedTenantId]);

  const resourcePolicyQuery = useQuery({
    ...queries.tenantResourcePolicy.get(selectedTenantId || ''),
    enabled: !!selectedTenantId,
  });

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

  const active = billingQuery.data?.currentSubscription;
  const upcoming = billingQuery.data?.upcomingSubscription;
  const plans = billingQuery.data?.plans;
  const coupons = billingQuery.data?.coupons;

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      const link = await cloudApi.organizationBillingPortalLinkGet(orgId);
      window.open(link.data.url, '_blank');
    } catch (e) {
      handleApiError(e as Parameters<typeof handleApiError>[0]);
    } finally {
      setPortalLoading(false);
    }
  };

  const subscriptionMutation = useMutation({
    mutationKey: ['organization:subscription:update', orgId],
    mutationFn: async ({ plan_code }: { plan_code: string }) => {
      const [plan, period] = plan_code.split('_');
      setLoading(plan_code);
      const response = await cloudApi.organizationSubscriptionUpdate(orgId, {
        plan,
        period,
      });
      return response.data;
    },
    onSuccess: async (data) => {
      if (data && 'checkoutUrl' in data) {
        window.location.href = data.checkoutUrl;
        return;
      }

      await queryClient.invalidateQueries({
        queryKey: ['organization-billing', orgId],
      });

      setLoading(undefined);
    },
    onError: handleApiError,
  });

  const activePlanCode = useMemo(() => {
    if (!active?.plan || active.plan === 'free') {
      return 'free';
    }
    return [active.plan, active.period].filter((x) => !!x).join('_');
  }, [active]);

  useEffect(() => {
    return setShowAnnual(active?.period?.includes('yearly') || false);
  }, [active]);

  const upcomingPlanCode = useMemo(() => {
    if (!upcoming?.plan) {
      return null;
    }
    return [upcoming.plan, upcoming.period].filter((x) => !!x).join('_');
  }, [upcoming]);

  const sortedPlans = useMemo(() => {
    return plans
      ?.filter(
        (v) =>
          ALLOWED_PLAN_CODES.includes(v.planCode) &&
          (v.planCode === 'free' ||
            (showAnnual
              ? v.period?.includes('yearly')
              : v.period?.includes('monthly'))),
      )
      .sort((a, b) => a.amountCents - b.amountCents);
  }, [plans, showAnnual]);

  const isUpgrade = useCallback(
    (plan: SubscriptionPlan) => {
      if (!active) {
        return true;
      }

      const activePlan = sortedPlans?.find(
        (p) => p.planCode === activePlanCode,
      );

      const activeAmount = activePlan?.amountCents || 0;

      return plan.amountCents > activeAmount;
    },
    [active, activePlanCode, sortedPlans],
  );

  const formattedEndDate = useMemo(() => {
    if (!active?.endsAt) {
      return null;
    }
    const date = new Date(active.endsAt);
    return date.toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }, [active?.endsAt]);

  const currentPlanDetails = useMemo(() => {
    if (!active?.plan) {
      return null;
    }
    return sortedPlans?.find((p) => p.planCode === activePlanCode);
  }, [active, activePlanCode, sortedPlans]);

  const enterpriseContactUrl = useMemo(() => {
    const baseUrl = 'https://cal.com/team/hatchet/website-demo';
    const notes = `Custom pricing request for organization '${organization.name}' (${orgId})`;
    return `${baseUrl}?notes=${encodeURIComponent(notes)}`;
  }, [organization.name, orgId]);

  const isDedicatedPlan = active?.plan === 'dedicated';

  if (billingQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Spinner />
      </div>
    );
  }

  if (billingQuery.isError) {
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
          await subscriptionMutation.mutateAsync({
            plan_code: isChangeConfirmOpen!.planCode,
          });
          setLoading(undefined);
          setChangeConfirmOpen(undefined);
        }}
        onCancel={() => setChangeConfirmOpen(undefined)}
        isLoading={!!loading}
      />
      <div className="space-y-6">
        {isDedicatedPlan ? (
          <div className="flex flex-row items-center justify-between">
            <p className="text-xl font-semibold leading-tight text-foreground">
              You are on the Dedicated plan
            </p>
            <Button
              onClick={manageClicked}
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
                onClick={manageClicked}
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
                          {new Intl.NumberFormat('en-US', {
                            style: 'currency',
                            currency: 'USD',
                          }).format(
                            currentPlanDetails.amountCents /
                              100 /
                              (currentPlanDetails.period === 'yearly' ? 12 : 1),
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
                        Switching to{' '}
                        {plans?.find(
                          (p) =>
                            p.planCode ===
                            [upcoming.plan, upcoming.period]
                              .filter((x) => !!x)
                              .join('_'),
                        )?.name || upcoming.plan}
                      </CardTitle>
                      <p className="text-sm text-muted-foreground">
                        This change will take effect on{' '}
                        {new Date(upcoming.startedAt).toLocaleDateString(
                          'en-US',
                          {
                            year: 'numeric',
                            month: 'long',
                            day: 'numeric',
                          },
                        )}
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
              {sortedPlans
                ?.filter(
                  (plan) =>
                    plan.planCode !== activePlanCode &&
                    plan.planCode !== upcomingPlanCode,
                )
                .map((plan, i) => (
                  <Card className="bg-muted/30 gap-4 flex-col flex" key={i}>
                    <CardHeader>
                      <CardTitle className="tracking-wide text-sm">
                        {plan.name}
                      </CardTitle>
                      <CardDescription className="py-4">
                        {new Intl.NumberFormat('en-US', {
                          style: 'currency',
                          currency: 'USD',
                        }).format(
                          plan.amountCents /
                            100 /
                            (plan.period === 'yearly' ? 12 : 1),
                        )}{' '}
                        per month billed {plan.period}*
                      </CardDescription>
                      <CardDescription>
                        <Button
                          disabled={loading === plan.planCode}
                          variant="default"
                          onClick={() => setChangeConfirmOpen(plan)}
                        >
                          {loading === plan.planCode ? (
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
