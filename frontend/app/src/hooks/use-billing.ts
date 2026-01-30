import api, { queries } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import {
  Organization,
  SubscriptionPlan,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
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

interface UseBillingOptions {
  organization: Organization;
}

export function useBilling({ organization }: UseBillingOptions) {
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});
  const orgId = organization.metadata.id;

  const [loading, setLoading] = useState<string>();
  const [showAnnual, setShowAnnual] = useState<boolean>(false);
  const [portalLoading, setPortalLoading] = useState(false);

  const billingQuery = useQuery({
    queryKey: ['organization-billing', orgId],
    queryFn: async () => {
      const result = await cloudApi.organizationBillingStateGet(orgId);
      return result.data;
    },
    enabled: !!orgId,
  });

  const active = billingQuery.data?.currentSubscription;
  const upcoming = billingQuery.data?.upcomingSubscription;
  const plans = billingQuery.data?.plans;
  const coupons = billingQuery.data?.coupons;

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

  const activePlanCode = useMemo(() => {
    if (!active?.plan || active.plan === 'free') {
      return 'free';
    }
    return [active.plan, active.period].filter((x) => !!x).join('_');
  }, [active]);

  const upcomingPlanCode = useMemo(() => {
    if (!upcoming?.plan) {
      return null;
    }
    return [upcoming.plan, upcoming.period].filter((x) => !!x).join('_');
  }, [upcoming]);

  useEffect(() => {
    setShowAnnual(active?.period?.includes('yearly') || false);
  }, [active]);

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

  const availablePlans = useMemo(() => {
    return sortedPlans?.filter(
      (plan) =>
        plan.planCode !== activePlanCode && plan.planCode !== upcomingPlanCode,
    );
  }, [sortedPlans, activePlanCode, upcomingPlanCode]);

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

  const currentPlanDetails = useMemo(() => {
    if (!active?.plan) {
      return null;
    }
    return sortedPlans?.find((p) => p.planCode === activePlanCode);
  }, [active, activePlanCode, sortedPlans]);

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

  const upcomingPlanName = useMemo(() => {
    if (!upcoming?.plan) {
      return null;
    }
    const upcomingCode = [upcoming.plan, upcoming.period]
      .filter((x) => !!x)
      .join('_');
    return (
      plans?.find((p) => p.planCode === upcomingCode)?.name || upcoming.plan
    );
  }, [upcoming, plans]);

  const upcomingStartDate = useMemo(() => {
    if (!upcoming?.startedAt) {
      return null;
    }
    return new Date(upcoming.startedAt).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }, [upcoming?.startedAt]);

  const enterpriseContactUrl = useMemo(() => {
    const baseUrl = 'https://cal.com/team/hatchet/website-demo';
    const notes = `Custom pricing request for organization '${organization.name}' (${orgId})`;
    return `${baseUrl}?notes=${encodeURIComponent(notes)}`;
  }, [organization.name, orgId]);

  const isDedicatedPlan = active?.plan === 'dedicated';

  const openBillingPortal = useCallback(async () => {
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
  }, [orgId, portalLoading, handleApiError]);

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

  const changePlan = useCallback(
    async (plan: SubscriptionPlan) => {
      await subscriptionMutation.mutateAsync({ plan_code: plan.planCode });
      setLoading(undefined);
    },
    [subscriptionMutation],
  );

  const formatPrice = useCallback((amountCents: number, period?: string) => {
    const monthly = amountCents / 100 / (period === 'yearly' ? 12 : 1);
    return new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD',
    }).format(monthly);
  }, []);

  return {
    isLoading: billingQuery.isLoading,
    isError: billingQuery.isError,
    changingPlanCode: loading,
    portalLoading,
    coupons,
    currentPlanDetails,
    formattedEndDate,
    isDedicatedPlan,
    upcoming,
    upcomingPlanName,
    upcomingStartDate,
    availablePlans,
    showAnnual,
    setShowAnnual,
    isUpgrade,
    enterpriseContactUrl,
    openBillingPortal,
    changePlan,
    formatPrice,
    activeTenants,
    detailedTenants,
    selectedTenantId,
    setSelectedTenantId,
    resourcePolicyQuery,
  };
}
