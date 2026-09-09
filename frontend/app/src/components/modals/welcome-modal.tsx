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
import { useMutation, useQuery } from '@tanstack/react-query';

const FREE_LIMIT_COPY: Record<string, { name: string; suffix?: string }> = {
  task_runs: { name: 'Task runs', suffix: ' daily' },
  task_runs_daily_limit: { name: 'Task runs', suffix: ' daily' },
  events: { name: 'External events', suffix: ' daily' },
  events_daily_limit: { name: 'External events', suffix: ' daily' },
  worker_slots_limit: { name: 'Concurrent runs' },
  users: { name: 'Users' },
};

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
  const { isControlPlaneEnabled, canBill } = useControlPlane();

  const welcomePlansQuery = useQuery({
    ...queries.controlPlane.subscriptionPlans(),
    enabled: open && isControlPlaneEnabled && canBill,
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
          plan: SubscriptionPlanCode.PayAsYouGo,
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
              The free tier includes everything you need to start building. No
              credit card, no time limit.
            </DialogDescription>
          </div>
          <Card
            variant="light"
            className="bg-transparent ring-1 ring-border/50 border-none"
          >
            <CardHeader className="p-4 border-b border-border/50">
              <CardTitle className="font-mono font-normal tracking-wider uppercase text-xs text-muted-foreground whitespace-nowrap">
                Included free
              </CardTitle>
            </CardHeader>
            <CardContent className="p-4">
              {welcomePlansQuery.isLoading ? (
                <div className="flex justify-center py-2">
                  <Spinner />
                </div>
              ) : (
                <ul className="space-y-2.5 text-sm">
                  {freeLimits?.map((fl) => {
                    const copy = FREE_LIMIT_COPY[fl.featureId];
                    return (
                      <li key={fl.featureId} className="flex justify-between">
                        <span className="text-muted-foreground">
                          {copy?.name ?? fl.name}
                        </span>
                        <span className="font-medium">
                          {fl.limit.toLocaleString()}
                          {copy?.suffix ?? ''}
                        </span>
                      </li>
                    );
                  })}
                </ul>
              )}
            </CardContent>
          </Card>
          <div className="flex w-full flex-col gap-2">
            <p className="text-sm text-muted-foreground">
              When you're ready for production, Pay as you Go removes these
              limits. There's no base monthly fee and you pay nothing until you
              scale past what's included for free.
            </p>
            <Button
              className="w-full"
              disabled={developerPlanMutation.isPending}
              onClick={() => {
                capture('welcome_modal_add_payment', {
                  tenant_id: tenantId,
                  organization_id: organizationId,
                  cta: 'upgrade_button',
                });
                developerPlanMutation.mutate();
              }}
            >
              {developerPlanMutation.isPending
                ? 'Redirecting…'
                : 'Upgrade to Pay as you Go – starts at $0/month'}
            </Button>
            <Button
              variant="ghost"
              className="w-full"
              onClick={() => {
                capture('welcome_modal_dismissed', {
                  tenant_id: tenantId,
                  organization_id: organizationId,
                  cta: 'start_building',
                });
                dismiss();
              }}
            >
              Start Building with these Limits
            </Button>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}
