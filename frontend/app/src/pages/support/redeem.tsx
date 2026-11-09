import { SetupCard, SetupScreen } from '@/components/layout/setup-card';
import { Alert, AlertDescription } from '@/components/v1/ui/alert';
import { Button } from '@/components/v1/ui/button';
import useControlPlane from '@/hooks/use-control-plane';
import { controlPlaneApi, controlPlaneMetaQuery } from '@/lib/api/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import { useApiError } from '@/lib/hooks';
import { initialSupportRedeemCode } from '@/lib/magic-link-bootstrap';
import { supportTokenQueryKey } from '@/lib/support-session';
import { accountSwitchMessage } from '@/pages/auth/magic-link/magic-link';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from '@tanstack/react-router';
import { useState } from 'react';

/**
 * Like a magic link: the code arrives in the URL fragment, is captured before
 * the app boots, and is consumed only on an explicit click so a link preview
 * or a strict-mode double effect cannot burn it.
 */
export default function SupportRedeem() {
  const code = initialSupportRedeemCode;
  const [errors, setErrors] = useState<string[]>(
    code ? [] : ['This support link is missing or invalid.'],
  );
  const { handleApiError } = useApiError({ setErrors });
  const navigate = useNavigate();
  const { isControlPlaneLoading } = useControlPlane();
  const { userGetCurrentQuery } = useUserApi();
  // Not before the control plane answers: asked while its status is unknown,
  // the user query goes to the shard, whose 403 rewrites the session cookie.
  const currentUser = useQuery({
    ...userGetCurrentQuery(),
    retry: false,
    enabled: !isControlPlaneLoading,
  });

  const redeem = useMutation({
    mutationKey: ['support-session:redeem'],
    mutationFn: async (redeemCode: string) =>
      (await controlPlaneApi.supportSessionRedeem({ code: redeemCode })).data,
    onSuccess: (session) => {
      // Anything fetched before belongs to a different identity. The control
      // plane status stays: without it the app runs as self-hosted for a tick.
      queryClient.removeQueries({
        predicate: (query) =>
          query.queryKey[0] !== controlPlaneMetaQuery.queryKey[0],
      });
      queryClient.setQueryData(
        supportTokenQueryKey(session.grant.tenantId),
        session,
      );
      navigate({
        to: appRoutes.tenantRunsRoute.to,
        params: { tenant: session.grant.tenantId },
        replace: true,
      });
    },
    onError: handleApiError,
  });

  if (errors.length > 0) {
    return (
      <SetupScreen>
        <SetupCard title="Support link unavailable">
          <p className="text-sm text-muted-foreground">
            Support links work once and expire after a few minutes. Run{' '}
            <code className="font-mono">hat support</code> again to open the
            tenant.
          </p>
          <Alert variant="destructive" className="mt-4">
            <AlertDescription>{errors.join(' ')}</AlertDescription>
          </Alert>
        </SetupCard>
      </SetupScreen>
    );
  }

  const warning = currentUser.isSuccess
    ? accountSwitchMessage(currentUser.data.email)
    : null;

  return (
    <SetupScreen>
      <SetupCard title="Open support session">
        <div className="flex flex-col gap-4">
          <p className="text-sm text-muted-foreground">
            This link can be used once. Continue only if you just ran{' '}
            <code className="font-mono">hat support</code>.
          </p>
          {warning && (
            <Alert>
              <AlertDescription>{warning}</AlertDescription>
            </Alert>
          )}
          <Button
            type="button"
            className="self-start"
            onClick={() => code && redeem.mutate(code)}
            disabled={redeem.isPending || currentUser.isLoading}
          >
            {redeem.isPending ? 'Opening…' : 'Open tenant'}
          </Button>
        </div>
      </SetupCard>
    </SetupScreen>
  );
}
