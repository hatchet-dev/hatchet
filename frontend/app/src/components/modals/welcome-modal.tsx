import { Button } from '@/components/v1/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useAnalytics } from '@/hooks/use-analytics';
import { queries } from '@/lib/api';
import { controlPlaneApi } from '@/lib/api/api';
import {
  SubscriptionPlanCode,
  SubscriptionPeriod,
  UpdateTenantSubscriptionResponse,
} from '@/lib/api/generated/control-plane/data-contracts';
import { ContentType } from '@/lib/api/generated/control-plane/http-client';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

const WELCOME_KEY = 'hatchet:show-welcome';

interface WelcomeModalProps {
  tenantId: string | undefined;
  open: boolean;
  onClose: () => void;
}

export function WelcomeModal({ tenantId, open, onClose }: WelcomeModalProps) {
  const { capture } = useAnalytics();
  const navigate = useNavigate();

  const welcomePlansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: open,
  });

  const developerPlanMutation = useMutation({
    mutationKey: ['welcome:developer-plan'],
    mutationFn: async () => {
      if (!tenantId) {
        throw new Error('No tenant id');
      }
      const response =
        await controlPlaneApi.request<UpdateTenantSubscriptionResponse>({
          path: `/api/v1/control-plane/billing/tenants/${tenantId}/subscription`,
          method: 'PATCH',
          body: {
            plan: SubscriptionPlanCode.Developer,
            period: SubscriptionPeriod.Monthly,
          },
          secure: true,
          type: ContentType.Json,
          format: 'json',
        });
      return response.data;
    },
    onSuccess: (data) => {
      localStorage.removeItem(WELCOME_KEY);
      onClose();
      if (data?.checkoutUrl) {
        window.location.href = data.checkoutUrl;
      }
    },
  });

  const dismiss = () => {
    localStorage.removeItem(WELCOME_KEY);
    onClose();
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(o) => {
        if (!o) {
          dismiss();
        }
      }}
    >
      <DialogContent className="max-w-lg text-center">
        <div className="flex flex-col items-center gap-6">
          <HatchetLogo variant="mark" className="h-10 w-10" />
          <div className="space-y-2">
            <DialogTitle className="text-center text-2xl">
              Welcome to Hatchet
            </DialogTitle>
            <DialogDescription className="text-center text-base text-muted-foreground">
              You&apos;re on the free plan with generous limits to get started.
              We&apos;ll let you know when you&apos;re getting close.
            </DialogDescription>
          </div>
          <ul className="w-full text-left text-base space-y-2.5 rounded-md border border-border/50 bg-muted/30 p-5">
            {welcomePlansQuery.isLoading ? (
              <li className="flex justify-center py-2">
                <Spinner />
              </li>
            ) : (
              welcomePlansQuery.data?.freeLimits?.map((fl) => (
                <li key={fl.featureId} className="flex justify-between">
                  <span className="text-muted-foreground">{fl.name}</span>
                  <span className="font-medium">
                    {fl.limit.toLocaleString()}
                  </span>
                </li>
              ))
            )}
          </ul>
          <div className="flex w-full flex-col gap-2">
            <Button
              className="w-full"
              disabled={developerPlanMutation.isPending}
              onClick={() => {
                capture('welcome_modal_add_payment', {
                  tenant_id: tenantId,
                  cta: 'developer_plan',
                });
                developerPlanMutation.mutate();
              }}
            >
              {developerPlanMutation.isPending ? (
                'Redirecting…'
              ) : (
                <>
                  Add a payment method to remove limits, no commitment required
                  &rarr;
                </>
              )}
            </Button>
            <Button
              variant="ghost"
              className="w-full"
              onClick={() => {
                capture('welcome_modal_dismissed', {
                  tenant_id: tenantId,
                  cta: 'continue_with_limits',
                });
                dismiss();
              }}
            >
              Continue with Limits
            </Button>
            <Button
              variant="ghost"
              className="w-full"
              onClick={() => {
                capture('welcome_modal_view_plans', {
                  tenant_id: tenantId,
                  cta: 'view_plan_options',
                });
                dismiss();
                if (tenantId) {
                  navigate({
                    to: appRoutes.tenantSettingsBillingRoute.to,
                    params: { tenant: tenantId },
                  });
                }
              }}
            >
              View Plan Options
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
