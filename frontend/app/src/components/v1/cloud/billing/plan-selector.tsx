import { isPayAsYouGoPlanCode } from './subscription-plan-code';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Spinner } from '@/components/v1/ui/loading';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import {
  Coupon,
  SubscriptionPlan,
  SubscriptionPlanFeatureGroup,
} from '@/lib/api/generated/control-plane/data-contracts';
import { CheckIcon, Cross2Icon } from '@radix-ui/react-icons';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useMemo } from 'react';

interface PlanSelectorProps {
  activePlanCode: string;
  activePlanAmountCents?: number;
  upcomingPlanCode: string | null;
  onSelectPlan: (plan: SubscriptionPlan) => void;
  enterpriseContactUrl: string;
  loading?: string;
  selectLabel?: string;
  coupons?: Coupon[];
}

function formatCurrency(cents: number, period?: string) {
  const monthly = period === 'yearly' ? cents / 100 / 12 : cents / 100;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(monthly);
}

function applyCoupon(amountCents: number, coupon: Coupon): number {
  if (coupon.percent) {
    return Math.round(amountCents * (1 - coupon.percent / 100));
  }
  if (coupon.amount_cents) {
    return Math.max(0, amountCents - coupon.amount_cents);
  }
  return amountCents;
}

function couponLabel(coupon: Coupon): string {
  if (coupon.percent) {
    return `${coupon.percent}% off`;
  }
  if (coupon.amount_cents) {
    return `${formatCurrency(coupon.amount_cents)} off`;
  }
  return coupon.name;
}

export function PlanSelector({
  activePlanCode,
  activePlanAmountCents,
  upcomingPlanCode,
  onSelectPlan,
  enterpriseContactUrl,
  loading,
  selectLabel,
  coupons,
}: PlanSelectorProps) {
  const { canBill, isControlPlaneEnabled } = useControlPlane();
  const activeCoupon = coupons?.[0];
  const plansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: isControlPlaneEnabled && canBill,
  });

  const plans = plansQuery.data?.plans;

  const sortedPlans = useMemo(() => {
    return plans
      ?.filter(
        (v) =>
          !v.legacy && v.planCode !== 'free' && v.planCode !== activePlanCode,
      )
      .sort((a, b) => a.amountCents - b.amountCents);
  }, [plans, activePlanCode]);

  const isUpgrade = useCallback(
    (plan: SubscriptionPlan) => {
      const activeAmount = activePlanAmountCents ?? 0;
      return (
        plan.amountCents > activeAmount ||
        (plan.amountCents === 0 && activeAmount === 0)
      );
    },
    [activePlanAmountCents],
  );

  const visiblePlans = sortedPlans;

  if (plansQuery.isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Spinner />
      </div>
    );
  }

  const cardCount = (visiblePlans?.length ?? 0) + 1;

  return (
    <div
      className={`grid grid-cols-1 gap-4 ${
        cardCount > 1 ? 'lg:grid-cols-5' : ''
      }`}
    >
      {visiblePlans?.map((plan) => {
        const isActive = plan.planCode === activePlanCode;
        const isUpcoming = plan.planCode === upcomingPlanCode;
        const usageBased =
          isPayAsYouGoPlanCode(plan.planCode) || plan.amountCents === 0;
        const hasCoupon = activeCoupon && plan.amountCents > 0;
        const discountedCents = hasCoupon
          ? applyCoupon(plan.amountCents, activeCoupon)
          : plan.amountCents;
        return (
          <PlanCard
            key={plan.planCode}
            name={plan.name}
            price={
              usageBased
                ? undefined
                : formatCurrency(discountedCents, plan.period)
            }
            usageBased={usageBased}
            recommended={usageBased}
            originalPrice={
              hasCoupon && discountedCents !== plan.amountCents
                ? formatCurrency(plan.amountCents, plan.period)
                : undefined
            }
            couponBadge={
              hasCoupon && discountedCents !== plan.amountCents
                ? couponLabel(activeCoupon)
                : undefined
            }
            featureGroups={plan.featureGroups}
            isUpgrade={isUpgrade(plan)}
            isActive={isActive}
            isUpcoming={isUpcoming}
            isLoading={loading === plan.planCode}
            onSelect={() => onSelectPlan(plan)}
            selectLabel={selectLabel}
            buttonLabel={usageBased ? 'Upgrade to Pay as you Go' : undefined}
            className={cardCount > 1 ? 'lg:col-span-3' : undefined}
          />
        );
      })}
      <PlanCard
        name="Enterprise"
        description="Have technical or compliance requirements?"
        enterpriseHighlights={[
          'Volume usage discounts',
          'Custom SLAs & uptime guarantees',
          'Dedicated support & onboarding',
          'SSO & audit logging',
          'Prometheus metrics',
          'HIPAA & BAAs',
          'VPC peering',
          'Bring your own cloud',
        ]}
        onSelect={() => window.open(enterpriseContactUrl, '_blank')}
        buttonLabel="Schedule a Sales Call"
        secondary
        className={cardCount > 1 ? 'lg:col-span-2' : undefined}
      />
    </div>
  );
}

function PlanCard({
  name,
  price,
  originalPrice,
  couponBadge,
  description,
  featureGroups,
  enterpriseHighlights,
  isUpgrade,
  isActive,
  isUpcoming,
  isLoading,
  onSelect,
  buttonLabel,
  selectLabel,
  usageBased,
  recommended,
  secondary,
  className,
}: {
  name: string;
  price?: string;
  originalPrice?: string;
  couponBadge?: string;
  description?: string;
  featureGroups?: SubscriptionPlanFeatureGroup[];
  enterpriseHighlights?: string[];
  isUpgrade?: boolean;
  isActive?: boolean;
  isUpcoming?: boolean;
  isLoading?: boolean;
  onSelect: () => void;
  buttonLabel?: string;
  selectLabel?: string;
  usageBased?: boolean;
  recommended?: boolean;
  secondary?: boolean;
  className?: string;
}) {
  return (
    <Card
      variant="light"
      className={`bg-transparent ring-1 border-none flex flex-col ${
        isActive || recommended ? 'ring-primary' : 'ring-border/50'
      } ${className ?? ''}`}
    >
      <CardHeader className="p-4 border-b border-border/50">
        <div className="flex items-center justify-between gap-2">
          <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
            {name}
          </CardTitle>
          {recommended ? (
            <Badge variant="successful" className="text-[10px]">
              Recommended
            </Badge>
          ) : null}
        </div>
      </CardHeader>
      <CardContent className="p-4 flex flex-col flex-1 gap-4">
        <div>
          {usageBased ? (
            <div className="space-y-1">
              <p className="text-xl font-semibold text-foreground">
                Pay only for what you use
              </p>
              <p className="text-sm text-muted-foreground">
                No monthly fee. Usage billed monthly.
              </p>
            </div>
          ) : price ? (
            <>
              {originalPrice && (
                <span className="text-sm text-muted-foreground line-through mr-2">
                  {originalPrice}
                </span>
              )}
              <span className="text-xl font-semibold text-foreground">
                {price}
              </span>
              <span className="text-xs text-muted-foreground ml-1">
                / mo + usage
              </span>
              {couponBadge && (
                <Badge
                  variant="successful"
                  className="ml-2 text-[10px] align-middle"
                >
                  {couponBadge}
                </Badge>
              )}
            </>
          ) : (
            <span className="text-sm text-muted-foreground">{description}</span>
          )}
        </div>

        {enterpriseHighlights && enterpriseHighlights.length > 0 && (
          <ul className="space-y-1.5 flex-1">
            {enterpriseHighlights.map((item) => (
              <li key={item} className="flex items-start gap-2 text-sm">
                <CheckIcon className="size-3.5 mt-0.5 shrink-0 text-primary" />
                <span className="text-foreground">{item}</span>
              </li>
            ))}
          </ul>
        )}

        {featureGroups && featureGroups.length > 0 && (
          <div className="space-y-3 flex-1">
            {featureGroups.map((group) => (
              <div key={group.name}>
                <p className="text-xs font-medium text-muted-foreground uppercase tracking-wide mb-1.5">
                  {group.name}
                </p>
                <ul className="space-y-1.5">
                  {group.features.map((f) => (
                    <li
                      key={f.featureId}
                      className={`flex items-start gap-2 text-sm ${!f.included ? 'opacity-40' : ''}`}
                    >
                      {f.included ? (
                        <CheckIcon className="size-3.5 mt-0.5 shrink-0 text-primary" />
                      ) : (
                        <Cross2Icon className="size-3.5 mt-0.5 shrink-0 text-muted-foreground" />
                      )}
                      <span>
                        <span
                          className={
                            f.included
                              ? 'text-foreground'
                              : 'text-muted-foreground'
                          }
                        >
                          {f.display?.primaryText ?? f.name}
                        </span>
                        {f.included && f.display?.secondaryText && (
                          <span className="text-foreground block">
                            {f.display.secondaryText}
                          </span>
                        )}
                      </span>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
          </div>
        )}

        <div className="mt-auto space-y-2">
          <Button
            variant={
              isActive || isUpcoming || secondary
                ? 'outline'
                : isUpgrade || usageBased
                  ? 'default'
                  : 'outline'
            }
            size="sm"
            disabled={isActive || isUpcoming || isLoading}
            onClick={onSelect}
            className="w-full"
          >
            {isLoading ? (
              <Spinner />
            ) : isActive ? (
              'Current Plan'
            ) : isUpcoming ? (
              'Upcoming Plan'
            ) : (
              buttonLabel ||
              selectLabel ||
              (isUpgrade ? 'Upgrade' : 'Downgrade')
            )}
          </Button>
          {usageBased ? (
            <p className="text-center text-xs text-muted-foreground">
              No monthly fee. Usage billed monthly. Cancel anytime.
            </p>
          ) : null}
        </div>
      </CardContent>
    </Card>
  );
}
