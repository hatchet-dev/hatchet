import { PlanSelector } from './plan-selector';
import { resolveSubscriptionPlanCode } from './subscription-plan-code';
import { usePylon } from '@/components/support-chat';
import { ConfirmDialog } from '@/components/v1/molecules/confirm-dialog';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Alert, AlertDescription, AlertTitle } from '@/components/v1/ui/alert';
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
import useControlPlane from '@/hooks/use-control-plane';
import { useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { controlPlaneApi } from '@/lib/api/api';
import {
  OrganizationBillingStateSubscription,
  SubscriptionPlan,
  SubscriptionPlanCode,
  SubscriptionPeriod,
  Coupon,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useApiError } from '@/lib/hooks';
import queryClient from '@/query-client';
import { useMutation, useQuery } from '@tanstack/react-query';
import React, { useEffect, useMemo, useState } from 'react';

interface SubscriptionProps {
  active?: OrganizationBillingStateSubscription;
  upcoming?: OrganizationBillingStateSubscription;
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

function formatPlanName(planCode?: string, plan?: string) {
  const value = planCode || plan;
  if (!value) {
    return 'Unknown plan';
  }

  return value
    .split('_')
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ');
}

function formatPeriod(period?: string) {
  if (!period) {
    return 'Active subscription';
  }

  return `${period.charAt(0).toUpperCase() + period.slice(1)} billing`;
}

function isLegacySubscriptionPlan(plan?: SubscriptionPlanCode) {
  return (
    plan === SubscriptionPlanCode.Starter ||
    plan === SubscriptionPlanCode.Growth
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object';
}

function getPlanChangeErrorMessage(error: unknown) {
  const fallback =
    'We could not change your plan. Please try again or contact us if this keeps happening.';

  if (isRecord(error)) {
    const response = error.response;
    if (isRecord(response)) {
      const data = response.data;
      if (isRecord(data)) {
        if (data.code === 'plan_already_attached') {
          return 'This plan is already attached to your organization. Refreshing billing details should show the current plan.';
        }

        if (typeof data.message === 'string') {
          return data.message;
        }

        if (typeof data.description === 'string') {
          return data.description;
        }
      }
    }
  }

  if (error instanceof Error && error.message) {
    return error.message;
  }

  return fallback;
}

const OFFICE_HOURS_URL = 'https://hatchet.run/office-hours';

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
  const [planChangeError, setPlanChangeError] = useState<string>();
  const [submittedPlanCode, setSubmittedPlanCode] = useState<string>();

  const { tenantId, tenant, billing, organizationId } = useTenantDetails();
  const { controlPlaneMeta, isControlPlaneEnabled } = useControlPlane();
  const { handleApiError } = useApiError({});
  const pylon = usePylon();
  const [portalLoading, setPortalLoading] = useState(false);
  const creditBalanceQuery = useQuery({
    ...queries.controlPlane.creditBalance(organizationId || ''),
    enabled:
      isControlPlaneEnabled && !!controlPlaneMeta?.canBill && !!organizationId,
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
      if (!organizationId) {
        return;
      }
      const link = await controlPlaneApi.billingPortalLinkGet(organizationId);
      window.open(link.data.url, '_blank');
    } catch (e) {
      handleApiError(e as any);
    } finally {
      setPortalLoading(false);
    }
  };

  const subscriptionMutation = useMutation({
    mutationKey: ['organization-subscription:update'],
    onMutate: ({ plan_code }: { plan_code: string }) => {
      setLoading(plan_code);
      setPlanChangeError(undefined);
      setSubmittedPlanCode(undefined);
    },
    mutationFn: async ({ plan_code }: { plan_code: string }) => {
      const [plan, period] = plan_code.split('_');
      if (!organizationId) {
        throw new Error('Organization not found for billing');
      }
      const response = await controlPlaneApi.organizationSubscriptionUpdate(
        organizationId,
        {
          plan: plan as SubscriptionPlanCode,
          period: period as SubscriptionPeriod,
        },
      );
      return response.data;
    },
    onSuccess: async (data, variables) => {
      if (data.checkoutUrl) {
        window.location.href = data.checkoutUrl;
        return;
      }

      setSubmittedPlanCode(variables.plan_code);

      const invalidations = [
        queryClient.invalidateQueries({
          queryKey: queries.controlPlane.billing(organizationId).queryKey,
        }),
      ];

      if (tenantId) {
        invalidations.push(
          queryClient.invalidateQueries({
            queryKey: queries.tenantResourcePolicy.get(tenantId).queryKey,
          }),
        );
      }

      await Promise.all(invalidations);
    },
    onError: (error) => {
      setPlanChangeError(getPlanChangeErrorMessage(error));
      setSubmittedPlanCode(undefined);
      setLoading(undefined);

      if (!isChangeConfirmOpen) {
        handleApiError(error as any);
      }

      if (organizationId) {
        void queryClient.invalidateQueries({
          queryKey: queries.controlPlane.billing(organizationId).queryKey,
        });
      }
    },
  });

  const activePlanCode = useMemo(() => {
    return resolveSubscriptionPlanCode(active, 'free') ?? 'free';
  }, [active]);

  useEffect(() => {
    return setShowAnnual(active?.period?.includes('yearly') || false);
  }, [active]);

  const upcomingPlanCode = useMemo(() => {
    return resolveSubscriptionPlanCode(upcoming, null);
  }, [upcoming]);

  useEffect(() => {
    if (!submittedPlanCode) {
      return;
    }

    if (
      activePlanCode !== submittedPlanCode &&
      upcomingPlanCode !== submittedPlanCode
    ) {
      return;
    }

    setLoading(undefined);
    setSubmittedPlanCode(undefined);
    setPlanChangeError(undefined);
    setChangeConfirmOpen(undefined);
  }, [activePlanCode, submittedPlanCode, upcomingPlanCode]);

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

  const currentPlanSummary = useMemo(() => {
    if (!active?.plan) {
      return null;
    }
    const plan = plans?.find((p) => p.planCode === activePlanCode);

    if (plan) {
      return {
        name: plan.name,
        amountCents: plan.amountCents,
        period: plan.period,
        legacy: !!plan.legacy,
      };
    }

    return {
      name: formatPlanName(activePlanCode, active.plan),
      period: active.period,
      legacy: isLegacySubscriptionPlan(active.plan),
    };
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

  const openChangeConfirm = (plan: SubscriptionPlan) => {
    setPlanChangeError(undefined);
    setSubmittedPlanCode(undefined);
    setChangeConfirmOpen(plan);
  };

  const closeChangeConfirm = () => {
    if (loading || submittedPlanCode) {
      return;
    }

    setPlanChangeError(undefined);
    setChangeConfirmOpen(undefined);
  };

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
            {planChangeError && (
              <Alert variant="destructive" className="mt-4">
                <AlertTitle>Plan change failed</AlertTitle>
                <AlertDescription>{planChangeError}</AlertDescription>
              </Alert>
            )}
          </>
        }
        submitLabel={'Change Plan'}
        onSubmit={() => {
          if (!isChangeConfirmOpen) {
            return;
          }

          subscriptionMutation.mutate({
            plan_code: isChangeConfirmOpen!.planCode,
          });
        }}
        onCancel={closeChangeConfirm}
        cancelDisabled={!!loading || !!submittedPlanCode}
        isLoading={!!loading}
      />

      <div>
        {isDedicatedPlan ? (
          <div className="flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
            <div className="space-y-1">
              <p className="text-xl font-semibold leading-tight text-foreground">
                You are on a Dedicated plan
              </p>
              <p className="text-sm text-muted-foreground">
                Contact us to make changes to your plan.
              </p>
            </div>
            <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
              {pylon.enabled && (
                <Button onClick={pylon.show} variant="outline">
                  Contact us
                </Button>
              )}
              <Button asChild variant="outline">
                <a href={OFFICE_HOURS_URL} target="_blank" rel="noreferrer">
                  Office hours
                </a>
              </Button>
              <Button
                onClick={manageClicked}
                variant="outline"
                disabled={portalLoading}
              >
                {portalLoading ? <Spinner /> : 'Manage Billing'}
              </Button>
            </div>
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

            {currentPlanSummary && (
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
                          {currentPlanSummary.name}
                        </span>
                        {currentPlanSummary.legacy && (
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
                      {typeof currentPlanSummary.amountCents === 'number' ? (
                        <>
                          <span className="text-2xl font-bold text-foreground">
                            {formatCurrency(
                              currentPlanSummary.amountCents,
                              currentPlanSummary.period,
                            )}
                          </span>
                          <span className="text-sm text-muted-foreground ml-1">
                            / month
                          </span>
                        </>
                      ) : (
                        <span className="text-sm text-muted-foreground">
                          {formatPeriod(currentPlanSummary.period)}
                        </span>
                      )}
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
                  openChangeConfirm(plan);
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
