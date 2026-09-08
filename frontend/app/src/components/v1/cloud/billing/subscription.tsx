import { formatInvoiceAmount, formatInvoiceDate } from './invoice-formatters';
import {
  isPayAsYouGoPlanCode,
  payAsYouGoPlan,
  resolveSubscriptionPlanCode,
} from './subscription-plan-code';
import { UpcomingInvoiceDialog } from './upcoming-invoice-dialog';
import { UpgradeGateDialog } from './upgrade-gate-dialog';
import { usePylon } from '@/components/support-chat';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { Badge } from '@/components/v1/ui/badge';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import { Spinner } from '@/components/v1/ui/loading';
import { Separator } from '@/components/v1/ui/separator';
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
  Coupon,
  OrganizationBillingStateSubscription,
  OrganizationInvoicePreview,
  SubscriptionPlan,
  SubscriptionPlanCode,
} from '@/lib/api/generated/control-plane/data-contracts';
import { OFFICE_HOURS_URL } from '@/lib/external-links';
import { useApiError } from '@/lib/hooks';
import { useQuery } from '@tanstack/react-query';
import React, { useMemo, useState } from 'react';

interface SubscriptionProps {
  active?: OrganizationBillingStateSubscription;
  upcoming?: OrganizationBillingStateSubscription;
  plans?: SubscriptionPlan[];
  coupons?: Coupon[];
  invoicePreviews?: OrganizationInvoicePreview[];
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
    plan === SubscriptionPlanCode.Growth ||
    plan === SubscriptionPlanCode.Developer ||
    plan === SubscriptionPlanCode.Team ||
    plan === SubscriptionPlanCode.Scale ||
    plan === SubscriptionPlanCode.Migration
  );
}

export const Subscription: React.FC<SubscriptionProps> = ({
  active,
  upcoming,
  plans,
  coupons,
  invoicePreviews,
}) => {
  const [invoicePreviewOpen, setInvoicePreviewOpen] = useState(false);
  const [upgradeOpen, setUpgradeOpen] = useState(false);

  const { tenantId, tenant, organizationId } = useTenantDetails();
  const { canBill, isControlPlaneEnabled } = useControlPlane();
  const { handleApiError } = useApiError({});
  const pylon = usePylon();
  const [portalLoading, setPortalLoading] = useState(false);
  const creditBalanceQuery = useQuery({
    ...queries.controlPlane.creditBalance(organizationId || ''),
    enabled: isControlPlaneEnabled && canBill && !!organizationId,
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

  const activePlanCode = useMemo(() => {
    return resolveSubscriptionPlanCode(active, 'free') ?? 'free';
  }, [active]);

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
  const nextInvoice = invoicePreviews?.[0];
  const isUsageBasedCurrentPlan = isPayAsYouGoPlanCode(activePlanCode);
  const showPlanSelector =
    !isDedicatedPlan && !isPayAsYouGoPlanCode(activePlanCode);

  return (
    <>
      <div>
        {isDedicatedPlan ? (
          <div className="space-y-6">
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
                <CardContent className="p-4 space-y-4">
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
                      {isUsageBasedCurrentPlan && nextInvoice ? (
                        <>
                          <span className="text-2xl font-bold text-foreground">
                            {formatInvoiceAmount(
                              nextInvoice.totalCents,
                              nextInvoice.currency,
                            )}
                          </span>
                          <p className="mt-1 text-sm text-muted-foreground">
                            estimated this period
                          </p>
                        </>
                      ) : typeof currentPlanSummary.amountCents === 'number' &&
                        currentPlanSummary.amountCents > 0 ? (
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
                      ) : isUsageBasedCurrentPlan ? (
                        <p className="text-sm text-muted-foreground">
                          Pay only for what you use
                        </p>
                      ) : (
                        <span className="text-sm text-muted-foreground">
                          {formatPeriod(currentPlanSummary.period)}
                        </span>
                      )}
                    </div>
                  </div>
                  {nextInvoice ? (
                    <div className="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
                      <p className="text-sm text-muted-foreground">
                        Next charge {formatInvoiceDate(nextInvoice.invoiceAt)}
                      </p>
                      <Button
                        variant="link"
                        size="sm"
                        className="h-auto p-0 justify-start sm:justify-end"
                        onClick={() => setInvoicePreviewOpen(true)}
                      >
                        View breakdown
                      </Button>
                    </div>
                  ) : null}
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

            {showPlanSelector && payAsYouGoPlan(plans) ? (
              <Card
                variant="light"
                className="bg-transparent ring-1 ring-border/50 border-none"
              >
                <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <div className="space-y-1">
                    <p className="text-sm font-medium text-foreground">
                      Upgrade to Pay as you Go
                    </p>
                    <p className="text-sm text-muted-foreground">
                      No monthly fee. Usage billed monthly.
                    </p>
                  </div>
                  <div className="flex flex-col gap-2 sm:flex-row">
                    <Button size="sm" onClick={() => setUpgradeOpen(true)}>
                      Upgrade
                    </Button>
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={() =>
                        window.open(
                          enterpriseContactUrl,
                          '_blank',
                          'noreferrer',
                        )
                      }
                    >
                      Talk to sales
                    </Button>
                  </div>
                </CardContent>
              </Card>
            ) : !isDedicatedPlan ? (
              <Card
                variant="light"
                className="bg-transparent ring-1 ring-border/50 border-none"
              >
                <CardContent className="flex flex-col gap-3 p-4 sm:flex-row sm:items-center sm:justify-between">
                  <p className="text-sm text-muted-foreground">
                    Need volume discounts, HIPAA, or VPC peering?
                  </p>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() =>
                      window.open(enterpriseContactUrl, '_blank', 'noreferrer')
                    }
                  >
                    Schedule a Sales Call
                  </Button>
                </CardContent>
              </Card>
            ) : null}
          </>
        )}
      </div>

      <UpcomingInvoiceDialog
        preview={nextInvoice ?? null}
        open={invoicePreviewOpen && !!nextInvoice}
        onOpenChange={setInvoicePreviewOpen}
      />

      {organizationId ? (
        <UpgradeGateDialog
          open={upgradeOpen}
          gate="usage"
          organizationId={organizationId}
          onDismiss={() => setUpgradeOpen(false)}
        />
      ) : null}
    </>
  );
};
