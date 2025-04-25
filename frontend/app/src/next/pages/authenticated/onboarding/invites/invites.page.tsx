import useUser from '@/next/hooks/use-user';
import { Button } from '@/next/components/ui/button';
import { Card, CardHeader, CardTitle } from '@/next/components/ui/card';
import { formatDistance } from 'date-fns';
import { Loader2 } from 'lucide-react';
import { DocsButton } from '@/next/components/ui/docs-button';
import docs from '@/next/lib/docs';
import { TenantBlock } from '../../dashboard/components/sidebar/user-dropdown';
import { TenantInvite } from '@/lib/api';

export function InviteCard({ invite }: { invite: TenantInvite }) {
  const { invites } = useUser();

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex flex-row gap-2 items-center justify-between">
          <div className="flex flex-row gap-4 items-center">
            <TenantBlock
              variant="default"
              tenant={{
                name: invite.tenantName || 'Team',
              }}
            />
          </div>
          <div className="flex flex-row gap-2 items-center">
            <span className="text-sm text-muted-foreground">
              Expires{' '}
              {formatDistance(new Date(invite.expires), new Date(), {
                addSuffix: true,
              })}
            </span>
            <Button
              variant="outline"
              onClick={() => invites.reject.mutate(invite.metadata.id)}
              disabled={invites.reject.isPending}
            >
              {invites.reject.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : null}
              Decline
            </Button>
            <Button
              onClick={async () => {
                await invites.accept.mutateAsync(invite.metadata.id);
                window.location.href = '/?tenant=' + invite.tenantId;
              }}
              disabled={invites.accept.isPending}
            >
              {invites.accept.isPending ? (
                <Loader2 className="h-4 w-4 mr-2 animate-spin" />
              ) : null}
              Accept
            </Button>
          </div>
        </CardTitle>
      </CardHeader>
    </Card>
  );
}

export default function InvitesPage() {
  const { invites } = useUser({ refetchInterval: 10 * 1000 });

  return (
    <div className="flex justify-center items-start py-12 px-4">
      <div className="max-w-xl w-full">
        <div className="flex flex-row justify-between items-center mb-4">
          <h1 className="text-2xl font-semibold leading-tight text-foreground">
            Team Invitations
          </h1>
          <DocsButton doc={docs.home.environments} size="icon" />
        </div>
        <p className="mb-6">
          You have been invited to join the following tenants. You can accept an
          invitation or create a new tenant.
        </p>

        {invites.loading ? (
          <div className="flex justify-center items-center py-12">
            <Loader2 className="h-8 w-8 animate-spin text-primary" />
          </div>
        ) : (
          <>
            {invites.list.length > 0 && (
              <div className="flex flex-col gap-4 mb-8">
                <h2 className="text-xl font-semibold">Pending Invites</h2>
                {invites.list.map((invite) => (
                  <InviteCard key={invite.metadata.id} invite={invite} />
                ))}
              </div>
            )}
          </>
        )}
      </div>
    </div>
  );
}
