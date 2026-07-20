import { InviteModal } from '@/components/modals/invite-modal';
import { pendingInvitesQuery } from '@/hooks/use-pending-invites';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { redirect, useNavigate } from '@tanstack/react-router';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function loader(_args: { request: Request }) {
  const { inviteCount } = await queryClient.fetchQuery(pendingInvitesQuery());

  if (inviteCount === 0) {
    throw redirect({ to: appRoutes.authenticatedRoute.to });
  }
}

export default function Invites() {
  const navigate = useNavigate();

  return (
    <div className="flex min-h-full flex-1 items-center justify-center">
      <InviteModal
        isOpen={true}
        onClose={() => navigate({ to: appRoutes.authenticatedRoute.to })}
      />
    </div>
  );
}
