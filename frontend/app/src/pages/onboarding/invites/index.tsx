import { SetupScreen } from '@/components/layout/setup-card';
import { InviteModal } from '@/components/modals/invite-modal';
import { Button } from '@/components/v1/ui/button';
import { pendingInvitesQuery } from '@/hooks/use-pending-invites';
import { fetchControlPlaneStatus } from '@/lib/api/api';
import { useUserApi } from '@/lib/api/user-wrapper';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { redirect, useNavigate } from '@tanstack/react-router';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function loader(_args: { request: Request }) {
  const { isControlPlaneEnabled } = await fetchControlPlaneStatus();

  const { inviteCount } = await queryClient.fetchQuery(
    pendingInvitesQuery(isControlPlaneEnabled),
  );

  if (inviteCount === 0) {
    throw redirect({ to: appRoutes.authenticatedRoute.to });
  }
}

export default function Invites() {
  const navigate = useNavigate();
  const { userUpdateLogoutMutation } = useUserApi();

  // The full-screen card replaces the top nav (and its account menu), so the
  // page carries its own sign-out, matching the other setup screens.
  const logoutMutation = useMutation({
    ...userUpdateLogoutMutation(),
    onSettled: () => {
      queryClient.clear();
      navigate({ to: appRoutes.authLoginRoute.to });
    },
  });

  return (
    <SetupScreen
      topRight={
        <Button
          variant="ghost"
          size="sm"
          onClick={() => logoutMutation.mutate()}
          disabled={logoutMutation.isPending}
        >
          Sign out
        </Button>
      }
    >
      <InviteModal
        variant="card"
        isOpen={true}
        onClose={() => navigate({ to: appRoutes.authenticatedRoute.to })}
      />
    </SetupScreen>
  );
}
