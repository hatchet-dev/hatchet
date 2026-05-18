import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Spinner } from '@/components/v1/ui/loading';
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
  showAnnual: boolean;
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
  showAnnual,
  onSelectPlan,
  enterpriseContactUrl,
  loading,
  selectLabel,
  coupons,
}: PlanSelectorProps) {
  const activeCoupon = coupons?.[0];
  const plansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
  });

  const plans = plansQuery.data?.plans;

  const sortedPlans = useMemo(() => {
    const nonLegacy = plans?.filter((v) => !v.legacy && v.planCode !== 'free');

    const hasYearlyVariant = (planCode: string) =>
      nonLegacy?.some(
        (p) =>
          p.planCode.startsWith(planCode.split('_')[0]) &&
          p.period?.includes('yearly'),
      );

    return nonLegacy
      ?.filter((v) => {
        if (showAnnual) {
          return v.period?.includes('yearly') || !hasYearlyVariant(v.planCode);
        }
        return v.period?.includes('monthly') || !v.period;
      })
      .sort((a, b) => a.amountCents - b.amountCents);
  }, [plans, showAnnual]);

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

  return (
    <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-4">
      {visiblePlans?.map((plan) => {
        const isActive = plan.planCode === activePlanCode;
        const isUpcoming = plan.planCode === upcomingPlanCode;
        const hasCoupon = activeCoupon && plan.amountCents > 0;
        const discountedCents = hasCoupon
          ? applyCoupon(plan.amountCents, activeCoupon)
          : plan.amountCents;
        return (
          <PlanCard
            key={plan.planCode}
            name={plan.name}
            price={formatCurrency(discountedCents, plan.period)}
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
            showAnnual={showAnnual}
            featureGroups={plan.featureGroups}
            isUpgrade={isUpgrade(plan)}
            isActive={isActive}
            isUpcoming={isUpcoming}
            isLoading={loading === plan.planCode}
            onSelect={() => onSelectPlan(plan)}
            selectLabel={selectLabel}
          />
        );
      })}
      <PlanCard
        name="Enterprise"
        description="Have technical or compliance requirements?"
        enterpriseHighlights={[
          '100M+ runs per month',
          'Custom SLAs & uptime guarantees',
          'Dedicated support & onboarding',
          'SSO / SAML & audit logging',
          'Bring your own cloud',
        ]}
        onSelect={() => window.open(enterpriseContactUrl, '_blank')}
        buttonLabel="Contact Us"
      />
    </div>
  );
}

function PlanCard({
  name,
  price,
  originalPrice,
  couponBadge,
  showAnnual,
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
}: {
  name: string;
  price?: string;
  originalPrice?: string;
  couponBadge?: string;
  showAnnual?: boolean;
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
}) {
  return (
    <Card
      variant="light"
      className={`bg-transparent ring-1 border-none flex flex-col ${
        isActive ? 'ring-primary' : 'ring-border/50'
      }`}
    >
      <CardHeader className="p-4 border-b border-border/50">
        <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
          {name}
        </CardTitle>
      </CardHeader>
      <CardContent className="p-4 flex flex-col flex-1 gap-4">
        <div>
          {price ? (
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
                / mo {!showAnnual ? ' + usage' : ''}
              </span>
              {couponBadge && (
                <Badge
                  variant="successful"
                  className="ml-2 text-[10px] align-middle"
                >
                  {couponBadge}
                </Badge>
              )}
              {showAnnual && (
                <p className="text-xs text-muted-foreground mt-1">
                  billed yearly + usage billed monthly
                </p>
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

        <Button
          variant={
            isActive || isUpcoming
              ? 'outline'
              : isUpgrade
                ? 'default'
                : 'outline'
          }
          size="sm"
          disabled={isActive || isUpcoming || isLoading}
          onClick={onSelect}
          className="w-full mt-auto"
        >
          {isLoading ? (
            <Spinner />
          ) : isActive ? (
            'Current Plan'
          ) : isUpcoming ? (
            'Upcoming Plan'
          ) : (
            buttonLabel || selectLabel || (isUpgrade ? 'Upgrade' : 'Downgrade')
          )}
        </Button>
      </CardContent>
    </Card>
  );
}
