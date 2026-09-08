import { useTenantDetails } from '@/hooks/use-tenant';
import { queries } from '@/lib/api';
import { controlPlaneApi } from '@/lib/api/api';
import {
  SubscriptionPeriod,
  SubscriptionPlanCode,
} from '@/lib/api/generated/control-plane/data-contracts';
import queryClient from '@/query-client';
import { useMutation } from '@tanstack/react-query';

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object';
}

export function getPlanChangeErrorMessage(error: unknown) {
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

export function useSubscriptionUpgrade(organizationId?: string | null) {
  const { tenantId } = useTenantDetails();

  return useMutation({
    mutationKey: ['organization-subscription:update', organizationId],
    mutationFn: async (planCode: string) => {
      const [plan, period] = planCode.split('_');
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
    onSuccess: async (data) => {
      if (data.checkoutUrl) {
        window.location.href = data.checkoutUrl;
        return;
      }

      const invalidations = [
        queryClient.invalidateQueries({
          queryKey: queries.controlPlane.billing(organizationId ?? '').queryKey,
        }),
        queryClient.invalidateQueries({
          queryKey: ['organization:entitlements:get', organizationId],
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
  });
}
