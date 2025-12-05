import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
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
import { Switch } from '@/components/v1/ui/switch';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import {
  TenantSubscription,
  SubscriptionPlan,
  Coupon,
} from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import queryClient from '@/query-client';
// import { ExclamationTriangleIcon } from '@radix-ui/react-icons';
import { useMutation } from '@tanstack/react-query';
import React, { useCallback, useEffect, useMemo, useState } from 'react';

interface SubscriptionProps {
  active?: TenantSubscription;
  plans?: SubscriptionPlan[];
  coupons?: Coupon[];
}

export const Subscription: React.FC<SubscriptionProps> = ({
  active,
  plans,
  coupons,
}) => {
  // Implement the logic for the Subscription component here

  const [loading, setLoading] = useState<string>();
  const [showAnnual, setShowAnnual] = useState<boolean>(false);
  const [isChangeConfirmOpen, setChangeConfirmOpen] = useState<
    SubscriptionPlan | undefined
  >(undefined);

  const { tenantId } = useCurrentTenantId();
  const { handleApiError } = useApiError({});
  const [portalLoading, setPortalLoading] = useState(false);

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      const link = await cloudApi.billingPortalLinkGet(tenantId);
      window.open(link.data.url, '_blank');
    } catch (e) {
      handleApiError(e as any);
    } finally {
      setPortalLoading(false);
    }
  };

  const subscriptionMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async ({ plan_code }: { plan_code: string }) => {
      const [plan, period] = plan_code.split(':');
      setLoading(plan_code);
      const response = await cloudApi.subscriptionUpsert(tenantId, { plan, period });
      return response.data;
    },
    onSuccess: async (data) => {
      // Check if response is a CheckoutURLResponse
      if (data && 'checkout_url' in data) {
        window.location.href = data.checkout_url;
        return;
      }

      // Otherwise it's a TenantSubscription, so invalidate queries
      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.tenantResourcePolicy.get(tenantId).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.cloud.billing(tenantId).queryKey,
        }),
      ]);

      setLoading(undefined);
    },
    onError: handleApiError,
  });

  const activePlanCode = useMemo(
    () => {
      if (!active?.plan || active.plan === 'free') {
        return 'free';
      }
      return [active.plan, active.period].filter((x) => !!x).join(':');
    },
    [active],
  );

  useEffect(() => {
    return setShowAnnual(active?.period?.includes('yearly') || false);
  }, [active]);

  const sortedPlans = useMemo(() => {
    return plans
      ?.filter(
        (v) =>
          v.plan_code === 'free' ||
          (showAnnual
            ? v.period?.includes('yearly')
            : v.period?.includes('monthly')),
      )
      .sort((a, b) => a.amount_cents - b.amount_cents);
  }, [plans, showAnnual]);

  const isUpgrade = useCallback(
    (plan: SubscriptionPlan) => {
      if (!active) {
        return true;
      }

      const activePlan = sortedPlans?.find(
        (p) => p.plan_code === activePlanCode,
      );

      const activeAmount = activePlan?.amount_cents || 0;

      return plan.amount_cents > activeAmount;
    },
    [active, activePlanCode, sortedPlans],
  );

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
            plan_code: isChangeConfirmOpen!.plan_code,
          });
          setLoading(undefined);
          setChangeConfirmOpen(undefined);
        }}
        onCancel={() => setChangeConfirmOpen(undefined)}
        isLoading={!!loading}
      />
      <div className="mx-auto py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <h3 className="text-xl font-semibold leading-tight text-foreground flex flex-row gap-2">
            Subscription
            {coupons?.map((coupon, i) => (
              <Badge key={`c${i}`} variant="successful">
                {coupon.name} coupon applied
              </Badge>
            ))}
          </h3>

          <div className="flex gap-4 items-center">
            <Button
              onClick={manageClicked}
              variant="outline"
              disabled={portalLoading}
            >
              {portalLoading ? <Spinner /> : 'Visit Billing Portal'}
            </Button>
            <div className="flex gap-2">
              <Switch
                id="sa"
                checked={showAnnual}
                onClick={() => {
                  setShowAnnual((checkedState) => !checkedState);
                }}
              />
              <Label htmlFor="sa" className="text-sm">
                Annual Billing{' '}
                <Badge variant="inProgress" className="ml-2">
                  Save up to 20%
                </Badge>
              </Label>
            </div>
          </div>
        </div>
        <p className="text-gray-700 dark:text-gray-300 my-4">
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
          <a href="https://hatchet.run/office-hours" className="underline">
            contact us
          </a>{' '}
          if you have custom requirements.
        </p>

        <div className="grid grid-cols-1 gap-4 sm:grid-cols-4">
          {sortedPlans?.map((plan, i) => (
            <Card className="bg-muted/30 gap-4 flex-col flex" key={i}>
              <CardHeader>
                <CardTitle className="tracking-wide text-sm">
                  {plan.name}
                </CardTitle>
                <CardDescription className="py-4">
                  $
                  {(
                    plan.amount_cents /
                    100 /
                    (plan.period == 'yearly' ? 12 : 1)
                  ).toLocaleString()}{' '}
                  per month billed {plan.period}*
                </CardDescription>
                <CardDescription>
                  <Button
                    disabled={
                      plan.plan_code === activePlanCode ||
                      loading === plan.plan_code
                    }
                    variant={
                      plan.plan_code !== activePlanCode ? 'default' : 'outline'
                    }
                    onClick={() => setChangeConfirmOpen(plan)}
                  >
                    {loading === plan.plan_code ? (
                      <Spinner />
                    ) : plan.plan_code === activePlanCode ? (
                      'Active'
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
                  onClick={() => window.open('https://hatchet.run/office-hours', '_blank')}
                >
                  Contact Us
                </Button>
              </CardDescription>
            </CardHeader>
          </Card>
        </div>
        {active?.note && <p className="mt-4">{active?.note}</p>}
        <p className="text-sm text-gray-500 mt-4">
          * subscription fee billed upfront {showAnnual ? 'yearly' : 'monthly'},
          overages billed at the end of each month for usage in that month
        </p>
      </div>
    </>
  );
};
