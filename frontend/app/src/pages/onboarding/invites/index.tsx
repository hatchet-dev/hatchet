import api, { queries } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation, useQuery } from '@tanstack/react-query';
import { useNavigate } from 'react-router-dom';
import { Loading } from '@/components/ui/loading';
import { Button } from '@/components/ui/button';

export default function TenantInvites() {
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});

  const listInvitesQuery = useQuery({
    ...queries.user.listInvites,
  });

  const acceptMutation = useMutation({
    mutationKey: ['tenant-invite:accept'],
    mutationFn: async (data: { invite: string }) => {
      await api.tenantInviteAccept(data);
    },
    onSuccess: async () => {
      await listInvitesQuery.refetch();
      // TODO: if there's more than 1 invite, stay on the screen
      navigate('/');
    },
    onError: handleApiError,
  });

  const rejectMutation = useMutation({
    mutationKey: ['tenant-invite:reject'],
    mutationFn: async (data: { invite: string }) => {
      await api.tenantInviteReject(data);
    },
    onSuccess: async () => {
      await listInvitesQuery.refetch();
      navigate('/');
    },
    onError: handleApiError,
  });

  if (listInvitesQuery.isLoading) {
    return <Loading />;
  }

  const invites = listInvitesQuery.data?.rows || [];

  const header =
    invites.length > 1 ? 'Join your team' : 'Join ' + invites[0].tenantName;

  return (
    <div className="flex flex-row flex-1 w-full h-full">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="lg:p-8 mx-auto w-screen">
          <div className="mx-auto flex w-40 flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                {header}
              </h1>
            </div>
            {invites.map((invite) => {
              return (
                <div
                  key={invite.metadata.id}
                  className="flex flex-col space-y-2 text-center"
                >
                  <p className="text-sm text-muted-foreground mb-4">
                    You got an invitation to join {invite.tenantName} on
                    Hatchet.
                  </p>
                  <div className="flex flex-row gap-2 justify-center">
                    <Button
                      variant="outline"
                      className="w-full"
                      onClick={() => {
                        rejectMutation.mutate({
                          invite: invite.metadata.id,
                        });
                      }}
                    >
                      Decline
                    </Button>
                    <Button
                      className="w-full"
                      onClick={() => {
                        acceptMutation.mutate({
                          invite: invite.metadata.id,
                        });
                      }}
                    >
                      Accept
                    </Button>
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      </div>
    </div>
  );
}
