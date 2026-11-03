import { Button } from '@/components/v1/ui/button';
import { useNow } from '@/hooks/use-now';
import { controlPlaneApi } from '@/lib/api/api';
import {
  clearSupportSession,
  getSupportSession,
  supportRoleLabel,
  useSupportSession,
} from '@/lib/support-session';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import {
  ArrowTopRightOnSquareIcon,
  ClockIcon,
  ExclamationTriangleIcon,
  LifebuoyIcon,
} from '@heroicons/react/24/outline';
import { useMutation } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';

function formatTimeLeft(expiresAt: string, now: number): string {
  const msLeft = Math.max(0, new Date(expiresAt).getTime() - now);
  const totalSeconds = Math.floor(msLeft / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return `${minutes}:${String(seconds).padStart(2, '0')}`;
}

const bannerClassName =
  'flex flex-wrap items-center justify-center gap-x-3 gap-y-1 border-b border-red-700 bg-red-600 px-4 py-1.5 text-center text-sm font-medium text-white dark:border-red-800 dark:bg-red-700';

const linkClassName =
  'inline-flex items-center gap-1 underline underline-offset-4 hover:text-white/80';

export function SupportSessionBanner() {
  const { session, isExpired } = useSupportSession();
  const now = useNow(1000);
  const navigate = useNavigate();

  const endMutation = useMutation({
    mutationKey: ['support-session:end'],
    mutationFn: async () => {
      // Read the token at call time: the renewal may have rotated it.
      const current = getSupportSession();
      if (!current) {
        return;
      }
      await controlPlaneApi.supportSessionEnd({
        headers: { Authorization: `Bearer ${current.token}` },
      });
    },
    onSettled: () => {
      clearSupportSession();
      queryClient.clear();
      navigate({ to: appRoutes.supportEndedRoute.to, replace: true });
    },
  });

  if (!session) {
    return null;
  }

  const { grant, consoleUrl } = session;

  if (isExpired) {
    return (
      <div role="alert" className={bannerClassName}>
        <ExclamationTriangleIcon className="h-4 w-4 flex-shrink-0" />
        <span>
          Support session expired — reopen the tenant from the support console
        </span>
        {consoleUrl && (
          <a
            href={consoleUrl}
            className={linkClassName}
            target="_blank"
            rel="noreferrer"
          >
            Open support console
            <ArrowTopRightOnSquareIcon className="h-3.5 w-3.5" />
          </a>
        )}
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
        {formatTimeLeft(session.expiresAt, now)} left
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
