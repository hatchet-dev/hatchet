import { Button } from '@/components/ui/button';
import api, { SubscriptionPlan, TenantSubscription, queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { TenantContextType } from '@/lib/outlet';
import queryClient from '@/query-client';
import { useMutation } from '@tanstack/react-query';
import React, { useCallback, useMemo } from 'react';
import { useOutletContext } from 'react-router-dom';

interface SubscriptionProps {
  active?: TenantSubscription;
  plans?: SubscriptionPlan[];
}

const Subscription: React.FC<SubscriptionProps> = ({ active, plans }) => {
  // Implement the logic for the Subscription component here
  const { tenant } = useOutletContext<TenantContextType>();

  const { handleApiError } = useApiError({});

  const subscriptionMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async ({ plan_code }: { plan_code: string }) => {
      const [plan, period] = plan_code.split(':');
      await api.subscriptionUpsert(tenant.metadata.id, { plan, period });
    },
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queries.tenantResourcePolicy.get(tenant.metadata.id).queryKey,
      });
    },
    onError: handleApiError,
  });

  const activePlanCode = useMemo(
    () => [active?.plan, active?.period].filter((x) => !!x).join(':'),
    [active],
  );

  const sortedPlans = useMemo(() => {
    return plans?.sort((a, b) => a.amount_cents - b.amount_cents);
  }, [plans]);

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
      </div>
      <p className="text-gray-700 dark:text-gray-300 my-4"></p>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
        {sortedPlans?.map((plan, i) => (
          <div className="flex flex-col" key={i}>
            {plan.plan_code}
            <p>
              ${plan.amount_cents / 100} billed {plan.period}
            </p>
            <Button
              disabled={plan.plan_code === activePlanCode}
              variant={
                plan.plan_code !== activePlanCode ? 'default' : 'outline'
              }
              onClick={async () => {
                await subscriptionMutation.mutateAsync({
                  plan_code: plan.plan_code,
                });
              }}
            >
              {plan.plan_code === activePlanCode
                ? 'Active'
                : isUpgrade(plan)
                  ? 'Upgrade'
                  : 'Downgrade'}
            </Button>
          </div>
        ))}
      </div>
      <p>{active?.note}</p>
    </div>
  );
};

export default Subscription;
