import { WELCOME_KEY } from './welcome-modal-state';
import { Button } from '@/components/v1/ui/button';
import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
} from '@/components/v1/ui/card';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from '@/components/v1/ui/dialog';
import { HatchetLogo } from '@/components/v1/ui/hatchet-logo';
import { Spinner } from '@/components/v1/ui/loading.tsx';
import { useAnalytics } from '@/hooks/use-analytics';
import useControlPlane from '@/hooks/use-control-plane';
import { queries } from '@/lib/api';
import { controlPlaneApi } from '@/lib/api/api';
import { SubscriptionPlanCode } from '@/lib/api/generated/control-plane/data-contracts';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

interface WelcomeModalProps {
  tenantId: string | undefined;
  organizationId: string | undefined;
  open: boolean;
  onClose: () => void;
}

export function WelcomeModal({
  tenantId,
  organizationId,
  open,
  onClose,
}: WelcomeModalProps) {
  const { capture } = useAnalytics();
  const navigate = useNavigate();
  const { controlPlaneCapabilities, isControlPlaneEnabled } = useControlPlane();

  const welcomePlansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled:
      open && isControlPlaneEnabled && !!controlPlaneCapabilities?.canBill,
  });

  const freeLimits = welcomePlansQuery.data?.freeLimits;

  const developerPlanMutation = useMutation({
    mutationKey: ['welcome:developer-plan'],
    mutationFn: async () => {
      if (!organizationId) {
        throw new Error('No organization id');
      }
      const response = await controlPlaneApi.organizationSubscriptionUpdate(
        organizationId,
        {
          plan: SubscriptionPlanCode.Developer,
        },
      );
      return response.data;
    },
    onSuccess: (data) => {
      localStorage.removeItem(WELCOME_KEY);
      onClose();
      if (data.checkoutUrl) {
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
      <DialogContent className="max-w-lg">
        <div className="flex w-full flex-col gap-6">
          <div className="flex flex-col gap-3">
            <HatchetLogo variant="mark" className="h-8 w-8" />
            <DialogTitle className="text-2xl font-semibold tracking-tight">
              Welcome to Hatchet
            </DialogTitle>
            <DialogDescription className="text-sm text-muted-foreground">
              You&apos;re on the free plan with daily limits.{' '}
              <button
                type="button"
                className="text-primary/70 underline underline-offset-4 hover:text-primary disabled:opacity-50"
                disabled={developerPlanMutation.isPending}
                onClick={() => {
                  capture('welcome_modal_add_payment', {
                    tenant_id: tenantId,
                    organization_id: organizationId,
                    cta: 'upgrade_link',
                  });
                  developerPlanMutation.mutate();
                }}
              >
                {developerPlanMutation.isPending ? 'Redirecting…' : 'Upgrade'}
              </button>{' '}
              to the pay-as-you-go (developer) plan to remove daily limits.
            </DialogDescription>
          </div>
          <Card
            variant="light"
            className="bg-transparent ring-1 ring-border/50 border-none"
          >
            <CardHeader className="p-4 border-b border-border/50">
              <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
                Free Plan Limits
              </CardTitle>
            </CardHeader>
            <CardContent className="p-4">
              {welcomePlansQuery.isLoading ? (
                <div className="flex justify-center py-2">
                  <Spinner />
                </div>
              ) : (
                <ul className="space-y-2.5 text-sm">
                  {freeLimits?.map((fl) => (
                    <li key={fl.featureId} className="flex justify-between">
                      <span className="text-muted-foreground">{fl.name}</span>
                      <span className="font-medium">
                        {fl.limit.toLocaleString()}
                      </span>
                    </li>
                  ))}
                </ul>
              )}
            </CardContent>
          </Card>
          <div className="flex w-full flex-col gap-2">
            <Button
              className="w-full"
              onClick={() => {
                capture('welcome_modal_dismissed', {
                  tenant_id: tenantId,
                  organization_id: organizationId,
                  cta: 'get_started',
                });
                dismiss();
              }}
            >
              Get started
            </Button>
            <Button
              variant="ghost"
              className="w-full"
              onClick={() => {
                capture('welcome_modal_view_plans', {
                  tenant_id: tenantId,
                  organization_id: organizationId,
                  cta: 'view_plan_options',
                });
                dismiss();
                if (tenantId) {
                  navigate({
                    to: appRoutes.organizationSettingsBillingRoute.to,
                    params: { organization: organizationId ?? '' },
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
