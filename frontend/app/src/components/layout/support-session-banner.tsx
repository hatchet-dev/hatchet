import { Button } from '@/components/v1/ui/button';
import { useNow } from '@/hooks/use-now';
import { useSupportSession } from '@/hooks/use-support-session';
import { useTenantDetails } from '@/hooks/use-tenant';
import { controlPlaneApi } from '@/lib/api/api';
import {
  markSupportSessionEnded,
  SUPPORT_GRANTS_QUERY_KEY,
  supportRoleLabel,
  supportTokenQueryKey,
} from '@/lib/support-session';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import {
  ClockIcon,
  ExclamationTriangleIcon,
  LifebuoyIcon,
} from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

function formatTimeLeft(expiresAt: string, now: number): string {
  const msLeft = Math.max(0, new Date(expiresAt).getTime() - now);
  const totalSeconds = Math.floor(msLeft / 1000);
  const hours = Math.floor(totalSeconds / 3600);
  const minutes = Math.floor((totalSeconds % 3600) / 60);
  const seconds = totalSeconds % 60;
  const mmss = `${minutes}:${String(seconds).padStart(2, '0')}`;
  return hours > 0
    ? `${hours}:${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`
    : mmss;
}

const bannerClassName =
  'flex flex-wrap items-center justify-center gap-x-3 gap-y-1 border-b border-red-700 bg-red-600 px-4 py-1.5 text-center text-sm font-medium text-white dark:border-red-800 dark:bg-red-700';

export function SupportSessionBanner() {
  const { tenant } = useTenantDetails();
  const { grant, isExpired } = useSupportSession(tenant?.metadata.id);
  const now = useNow(1000);
  const navigate = useNavigate();

  const endMutation = useMutation({
    mutationKey: ['support-session:end'],
    mutationFn: async () => {
      if (!grant) {
        return;
      }
      await controlPlaneApi.supportSessionEnd({ grantId: grant.id });
    },
    onSettled: () => {
      if (grant) {
        markSupportSessionEnded(grant);
        queryClient.removeQueries({
          queryKey: supportTokenQueryKey(grant.tenantId),
        });
      }
      void queryClient.invalidateQueries({
        queryKey: SUPPORT_GRANTS_QUERY_KEY,
      });
      navigate({ to: appRoutes.supportEndedRoute.to, replace: true });
    },
  });

  if (!grant) {
    return null;
  }

  if (isExpired) {
    return (
      <div role="alert" className={bannerClassName}>
        <ExclamationTriangleIcon className="h-4 w-4 flex-shrink-0" />
        <span>
          Support session expired — run{' '}
          <code className="font-mono font-normal">hat support</code> again to
          reopen the tenant
        </span>
      </div>
    );
  }

  return (
    <div role="alert" className={bannerClassName}>
      <LifebuoyIcon className="h-4 w-4 flex-shrink-0" />
      <span>Support session</span>
      <span aria-hidden="true">·</span>
      <span className="max-w-[16rem] truncate" title={grant.tenantName}>
        {grant.tenantName}
      </span>
      <span aria-hidden="true">·</span>
      <span>{supportRoleLabel(grant.role)}</span>
      <span aria-hidden="true">·</span>
      <span className="max-w-[24rem] truncate font-normal" title={grant.reason}>
        {grant.reason}
      </span>
      <span aria-hidden="true">·</span>
      <span className="inline-flex items-center gap-1 tabular-nums">
        <ClockIcon className="h-4 w-4" />
        {formatTimeLeft(grant.expiresAt, now)} left
      </span>
      <Button
        type="button"
        variant="outline"
        size="sm"
        className="h-6 border-white/40 bg-transparent px-2 text-xs text-white hover:bg-white/10 hover:text-white"
        onClick={() => endMutation.mutate()}
        disabled={endMutation.isPending}
      >
        {endMutation.isPending ? 'Ending…' : 'End session'}
      </Button>
    </div>
  );
}
