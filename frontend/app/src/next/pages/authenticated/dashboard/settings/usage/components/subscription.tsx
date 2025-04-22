import {
  Alert,
  AlertDescription,
  AlertTitle,
} from '@/next/components/ui/alert';
import { Badge } from '@/next/components/ui/badge';
import { Button } from '@/next/components/ui/button';
import {
  Card,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/next/components/ui/card';
import { Label } from '@/next/components/ui/label';
import { Switch } from '@/next/components/ui/switch';
import { SubscriptionPlan } from '@/lib/api/generated/cloud/data-contracts';
import useBilling, { Plan } from '@/next/hooks/use-billing';
import { ExclamationTriangleIcon } from '@radix-ui/react-icons';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { DestructiveDialog } from '@/next/components/ui/dialog/destructive-dialog';
import { ROUTES } from '@/next/lib/routes';

export const Subscription: React.FC = () => {
  // Implement the logic for the Subscription component here

  const { billing } = useBilling();

  const active = useMemo(() => billing.state?.subscription, [billing.state]);
  const plans = useMemo(() => billing.state?.plans, [billing.state]);
  const coupons = useMemo(() => billing.state?.coupons, [billing.state]);

  const [showAnnual, setShowAnnual] = useState<boolean>(false);
  const [isChangeConfirmOpen, setChangeConfirmOpen] = useState<
    SubscriptionPlan | undefined
  >(undefined);

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
    <>
      <DestructiveDialog
        open={!!isChangeConfirmOpen}
        onOpenChange={(open) => {
          if (!open) {
            setChangeConfirmOpen(undefined);
          }
        }}
        title={'Confirm Change Plan'}
        submitVariant="default"
        hideAlert={true}
        requireTextConfirmation={false}
        confirmationText={isChangeConfirmOpen?.name || ''}
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
        confirmButtonText={'Change Plan'}
        onConfirm={async () => {
          await billing.changePlan.mutateAsync({
            plan_code: isChangeConfirmOpen!.plan_code as Plan,
          });
          setChangeConfirmOpen(undefined);
        }}
        onCancel={() => setChangeConfirmOpen(undefined)}
        isLoading={billing.changePlan.isPending}
      />
      <div className="flex flex-row justify-between items-center">
        <h3 className="text-xl font-semibold leading-tight text-foreground flex flex-row gap-2">
          Subscription
          {coupons?.map((coupon, i) => (
            <Badge key={`c${i}`} variant="successful">
              {coupon.name} coupon applied
            </Badge>
          ))}
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
            <Badge variant="successful" className="ml-2">
              Save up to 20%
            </Badge>
          </Label>
        </div>
      </div>
      <p className="text-gray-700 dark:text-gray-300 my-4">
        For plan details, please visit{' '}
        <a
          href={ROUTES.common.pricing}
          className="underline"
          target="_blank"
          rel="noreferrer"
        >
          our pricing page
        </a>{' '}
        or{' '}
        <a
          href={ROUTES.common.contact}
          className="underline"
          target="_blank"
          rel="noreferrer"
        >
          contact us
        </a>{' '}
        if you have custom requirements.
      </p>
      {!billing.hasPaymentMethods && (
        <Alert variant="warning" className="mb-4">
          <ExclamationTriangleIcon className="h-4 w-4" />
          <AlertTitle className="font-semibold">No Payment Method.</AlertTitle>
          <AlertDescription>
            A payment method is required to upgrade your subscription, please{' '}
            <a
              href={billing.getManagedUrl.data}
              className="underline pointer"
              target="_blank"
              rel="noreferrer"
            >
              add one
            </a>{' '}
            first.
          </AlertDescription>
        </Alert>
      )}

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-3">
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
                  loading={billing.changePlan.isPending}
                  disabled={
                    !billing.hasPaymentMethods ||
                    plan.plan_code === activePlanCode ||
                    billing.changePlan.isPending
                  }
                  variant={
                    plan.plan_code !== activePlanCode ? 'default' : 'outline'
                  }
                  onClick={() => setChangeConfirmOpen(plan)}
                >
                  {plan.plan_code === activePlanCode
                    ? 'Active'
                    : isUpgrade(plan)
                      ? 'Upgrade'
                      : 'Downgrade'}
                </Button>
              </CardDescription>
            </CardHeader>
          </Card>
        ))}
      </div>
      {active?.note && <p className="mt-4">{active?.note}</p>}
      <p className="text-sm text-gray-500 mt-4">
        * subscription fee billed upfront {showAnnual ? 'yearly' : 'monthly'},
        overages billed at the end of each month for usage in that month
      </p>
    </>
  );
};
