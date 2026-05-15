import { PlanSelector } from './plan-selector';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Label } from '@/components/v1/ui/label';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
import { Switch } from '@/components/v1/ui/switch';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/components/v1/ui/tooltip';
import { useCurrentTenantId, useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { controlPlaneApi } from '@/lib/api/api';
import {
  TenantSubscription,
  SubscriptionPlan,
  SubscriptionPlanCode,
  SubscriptionPeriod,
  Coupon,
  UpdateTenantSubscriptionResponse,
} from '@/lib/api/generated/control-plane/data-contracts';
import { ContentType } from '@/lib/api/generated/control-plane/http-client';
import { useApiError } from '@/lib/hooks';
import queryClient from '@/query-client';
import { useMutation, useQuery } from '@tanstack/react-query';
import React, { useEffect, useMemo, useState } from 'react';

interface SubscriptionProps {
  active?: TenantSubscription;
  upcoming?: TenantSubscription;
  plans?: SubscriptionPlan[];
  coupons?: Coupon[];
}

function formatCurrency(cents: number, period?: string) {
  const monthly = period === 'yearly' ? cents / 100 / 12 : cents / 100;
  return new Intl.NumberFormat('en-US', {
    style: 'currency',
    currency: 'USD',
  }).format(monthly);
}

export const Subscription: React.FC<SubscriptionProps> = ({
  active,
  upcoming,
  plans,
  coupons,
}) => {
  const [loading, setLoading] = useState<string>();
  const [showAnnual, setShowAnnual] = useState<boolean>(false);
  const [isChangeConfirmOpen, setChangeConfirmOpen] = useState<
    SubscriptionPlan | undefined
  >(undefined);

  const { tenantId } = useCurrentTenantId();
  const { tenant, billing } = useTenantDetails();
  const { handleApiError } = useApiError({});
  const [portalLoading, setPortalLoading] = useState(false);
  const creditBalanceQuery = useQuery({
    ...queries.controlPlane.creditBalance(tenantId),
  });

  const creditBalance = useMemo(() => {
    const balanceCents = creditBalanceQuery.data?.balanceCents ?? 0;

    if (balanceCents >= 0) {
      return null;
    }

    const currencyCode = (creditBalanceQuery.data?.currency || 'USD')
      .toUpperCase()
      .slice(0, 3);
    let formatted: string;
    try {
      formatted = new Intl.NumberFormat('en-US', {
        style: 'currency',
        currency: currencyCode,
      }).format(Math.abs(balanceCents) / 100);
    } catch {
      formatted = `$${(Math.abs(balanceCents) / 100).toFixed(2)}`;
    }

    const description = creditBalanceQuery.data?.description?.trim();
    const expires = creditBalanceQuery.data?.expiresAt;

    return {
      amount: formatted,
      description,
      expires,
    };
  }, [
    creditBalanceQuery.data?.balanceCents,
    creditBalanceQuery.data?.currency,
    creditBalanceQuery.data?.description,
    creditBalanceQuery.data?.expiresAt,
  ]);

  const manageClicked = async () => {
    try {
      if (portalLoading) {
        return;
      }
      setPortalLoading(true);
      const link = await controlPlaneApi.request<{ url?: string }>({
        path: `/api/v1/control-plane/billing/tenants/${tenantId}/billing-portal-link`,
        method: 'GET',
        secure: true,
        format: 'json',
      });
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
      const [plan, period] = plan_code.split('_');
      setLoading(plan_code);
      const response =
        await controlPlaneApi.request<UpdateTenantSubscriptionResponse>({
          path: `/api/v1/control-plane/billing/tenants/${tenantId}/subscription`,
          method: 'PATCH',
          body: {
            plan: plan as SubscriptionPlanCode,
            period: period as SubscriptionPeriod,
          },
          secure: true,
          type: ContentType.Json,
          format: 'json',
        });
      return response.data;
    },
    onSuccess: async (data) => {
      if (data?.checkoutUrl) {
        window.location.href = data.checkoutUrl;
        return;
      }

      await Promise.all([
        queryClient.invalidateQueries({
          queryKey: queries.tenantResourcePolicy.get(tenantId).queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: queries.controlPlane.billing(tenantId).queryKey,
        }),
      ]);

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

  const activePlanAmountCents = useMemo(
    () => plans?.find((p) => p.planCode === activePlanCode)?.amountCents,
    [plans, activePlanCode],
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
    return plans?.find((p) => p.planCode === activePlanCode);
  }, [active, activePlanCode, plans]);

  const enterpriseContactUrl = useMemo(() => {
    const baseUrl = 'https://cal.com/team/hatchet/website-demo';
    if (!tenant) {
      return baseUrl;
    }
    const tenantName = tenant.name || 'Unknown';
    const tenantUuid = tenant.metadata?.id || tenantId;
    const notes = `Custom pricing request for tenant '${tenantName}' (${tenantUuid})`;
    return `${baseUrl}?notes=${encodeURIComponent(notes)}`;
  }, [tenant, tenantId]);

  const isDedicatedPlan = active?.plan === 'dedicated';

  return (
    <>
      <ConfirmDialog
        isOpen={!!isChangeConfirmOpen}
        title={'Confirm Plan Change'}
        submitVariant="default"
        description={
          <>
            Are you sure you'd like to change to the{' '}
            <span className="font-semibold">{isChangeConfirmOpen?.name}</span>{' '}
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

      <div>
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
              {portalLoading ? <Spinner /> : 'Manage Billing'}
            </Button>
          </div>
        ) : (
          <>
            <h3 className="flex flex-row items-center gap-2 text-xl font-semibold leading-tight text-foreground">
              Subscription
              {coupons?.map((coupon, i) => (
                <Badge key={`c${i}`} variant="successful">
                  {coupon.name} coupon applied
                </Badge>
              ))}
            </h3>

            <Separator className="my-4" />

            {creditBalance && (
              <Card
                variant="light"
                className="mb-6 bg-transparent ring-1 ring-emerald-500/30 border-none"
              >
                <CardHeader className="p-4">
                  <div className="flex items-center justify-between gap-4">
                    <div>
                      <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
                        Available Credit
                      </CardTitle>
                      <p className="mt-1 text-sm text-muted-foreground">
                        {creditBalance.description ||
                          'Applied to upcoming invoices.'}
                      </p>
                    </div>
                    <div className="text-right">
                      <div className="text-xl font-semibold text-foreground whitespace-nowrap">
                        {creditBalance.amount}
                      </div>
                      {creditBalance.expires && (
                        <p className="mt-1 text-xs text-muted-foreground whitespace-nowrap">
                          Expires{' '}
                          <RelativeDate date={creditBalance.expires} future />
                        </p>
                      )}
                    </div>
                  </div>
                </CardHeader>
              </Card>
            )}

            {currentPlanDetails && (
              <Card
                variant="light"
                className="mb-6 bg-transparent ring-1 ring-border/50 border-none"
              >
                <CardHeader className="p-4 border-b border-border/50 flex flex-row items-center justify-between">
                  <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground">
                    Current Plan
                  </CardTitle>
                  <Button
                    onClick={manageClicked}
                    variant="outline"
                    size="sm"
                    disabled={portalLoading}
                  >
                    {portalLoading ? <Spinner /> : 'Manage Billing'}
                  </Button>
                </CardHeader>
                <CardContent className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="flex items-center gap-2">
                        <span className="text-lg font-semibold text-foreground">
                          {currentPlanDetails.name}
                        </span>
                        {currentPlanDetails.legacy && (
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger>
                                <Badge variant="queued">Legacy</Badge>
                              </TooltipTrigger>
                              <TooltipContent side="right">
                                You're on a legacy plan which is no longer
                                offered. Contact us if you have any questions.
                              </TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        )}
                      </div>
                      {formattedEndDate && (
                        <p className="mt-1 text-sm text-muted-foreground">
                          Service ends on {formattedEndDate}
                        </p>
                      )}
                    </div>
                    <div className="text-right">
                      <span className="text-2xl font-bold text-foreground">
                        {formatCurrency(
                          currentPlanDetails.amountCents,
                          currentPlanDetails.period,
                        )}
                      </span>
                      <span className="text-sm text-muted-foreground ml-1">
                        / month
                      </span>
                    </div>
                  </div>
                </CardContent>
              </Card>
            )}

            {upcoming && upcoming.plan && (
              <Card
                variant="light"
                className="mb-6 bg-transparent ring-1 ring-yellow-500/30 border-none"
              >
                <CardHeader className="p-4">
                  <div className="flex items-center justify-between">
                    <div>
                      <div className="flex items-center gap-2 mb-1">
                        <Badge variant="inProgress">Scheduled Change</Badge>
                      </div>
                      <p className="text-sm text-foreground">
                        Switching to{' '}
                        <span className="font-semibold">
                          {plans?.find(
                            (p) =>
                              p.planCode ===
                              [upcoming.plan, upcoming.period]
                                .filter((x) => !!x)
                                .join('_'),
                          )?.name || upcoming.plan}
                        </span>
                      </p>
                      <p className="text-xs text-muted-foreground mt-0.5">
                        Takes effect on{' '}
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

            <div className="flex flex-row items-center justify-between mb-4">
              <p className="text-sm text-muted-foreground">
                For plan details, visit{' '}
                <a
                  href="https://hatchet.run/pricing"
                  className="text-primary/70 hover:text-primary hover:underline"
                  target="_blank"
                  rel="noreferrer"
                >
                  our pricing page
                </a>{' '}
                or{' '}
                <a
                  href="https://hatchet.run/office-hours"
                  className="text-primary/70 hover:text-primary hover:underline"
                >
                  contact us
                </a>{' '}
                for custom requirements.
              </p>

              <div className="flex gap-2 items-center shrink-0 ml-4">
                <Switch
                  id="sa"
                  checked={showAnnual}
                  onClick={() => {
                    setShowAnnual((checkedState) => !checkedState);
                  }}
                />
                <Label htmlFor="sa" className="text-sm whitespace-nowrap">
                  Annual Billing
                  <Badge variant="inProgress" className="ml-2">
                    Save up to 20%
                  </Badge>
                </Label>
              </div>
            </div>

            <PlanSelector
              activePlanCode={activePlanCode}
              activePlanAmountCents={activePlanAmountCents}
              upcomingPlanCode={upcomingPlanCode}
              showAnnual={showAnnual}
              onSelectPlan={(plan) => {
                if (!billing?.hasPaymentMethods) {
                  subscriptionMutation.mutate({ plan_code: plan.planCode });
                } else {
                  setChangeConfirmOpen(plan);
                }
              }}
              enterpriseContactUrl={enterpriseContactUrl}
              loading={loading}
              coupons={coupons}
            />
          </>
        )}
      </div>
    </>
  );
};
