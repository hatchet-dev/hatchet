import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { Badge } from '@/components/ui/badge';
import { Button } from '@/components/ui/button';
import { Label } from '@/components/ui/label';
import { Spinner } from '@/components/ui/loading';
import { Switch } from '@/components/ui/switch';
import api, { SubscriptionPlan, TenantSubscription, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { TenantContextType } from '@/lib/outlet';
import queryClient from '@/query-client';
import { ExclamationTriangleIcon } from '@radix-ui/react-icons';
import { useMutation } from '@tanstack/react-query';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useOutletContext } from 'react-router-dom';

interface SubscriptionProps {
  active?: TenantSubscription;
  plans?: SubscriptionPlan[];
  hasPaymentMethods?: boolean;
}

const Subscription: React.FC<SubscriptionProps> = ({
  active,
  plans,
  hasPaymentMethods,
}) => {
  // Implement the logic for the Subscription component here

  const [loading, setLoading] = useState<string>();
  const [showAnnual, setShowAnnual] = useState<boolean>(false);

  const { tenant } = useOutletContext<TenantContextType>();
  const { handleApiError } = useApiError({});
  const [portalLoading, setPortalLoading] = useState(false);

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      const link = await api.billingPortalLinkGet(tenant.metadata.id);
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
      await api.subscriptionUpsert(tenant.metadata.id, { plan, period });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({
        queryKey: queries.tenantResourcePolicy.get(tenant.metadata.id).queryKey,
      });
      setLoading(undefined);
    },
    onError: handleApiError,
  });

  const activePlanCode = useMemo(
    () =>
      active?.plan
        ? [active.plan, active.period].filter((x) => !!x).join(':')
        : 'free',
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
    <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground">
          Subscription
        </h3>
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
      {!hasPaymentMethods && (
        <Alert variant="destructive" className="mb-4">
          <ExclamationTriangleIcon className="h-4 w-4" />
          <AlertTitle className="font-semibold">No Payment Method.</AlertTitle>
          <AlertDescription>
            A payment method is required to upgrade your subscription, please{' '}
            <a onClick={manageClicked} className="underline pointer" href="#">
              add one
            </a>{' '}
            first.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {sortedPlans?.map((plan, i) => (
          <div className="flex flex-col" key={i}>
            <b>{plan.name}</b>
            <p>
              ${(plan.amount_cents / 100).toLocaleString()} billed {plan.period}
              *
            </p>
            <Button
              disabled={
                !hasPaymentMethods ||
                plan.plan_code === activePlanCode ||
                loading === plan.plan_code
              }
              variant={
                plan.plan_code !== activePlanCode ? 'default' : 'outline'
              }
              onClick={async () => {
                await subscriptionMutation.mutateAsync({
                  plan_code: plan.plan_code,
                });
              }}
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
          </div>
        ))}
      </div>
      {active?.note && <p className="mt-4">{active?.note}</p>}
      <p className="text-sm text-gray-500 mt-4">
        * subscription fee billed upfront {showAnnual ? 'yearly' : 'monthly'},
        overages billed monthly in arrears
      </p>
    </div>
  );
};

export default Subscription;