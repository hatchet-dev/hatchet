import api, { TenantVersion } from '@/lib/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import {
  LoaderFunctionArgs,
  redirect,
  useLoaderData,
  useNavigate,
} from 'react-router-dom';
import { Button } from '@/components/ui/button';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function loader(_args: LoaderFunctionArgs) {
  const res = await api.userListTenantInvites();

  const invites = res.data.rows || [];

  if (invites.length == 0) {
    throw redirect('/');
  }

  return {
    invites,
  };
}

export default function TenantInvites() {
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});

  const { invites } = useLoaderData() as Awaited<ReturnType<typeof loader>>;

  const acceptMutation = useMutation({
    mutationKey: ['tenant-invite:accept'],
    mutationFn: async (data: {
      tenantId: string;
      inner: { invite: string };
    }) => {
      await api.tenantInviteAccept(data.inner);
      return data.tenantId;
    },
    onSuccess: async (tenantId: string) => {
      try {
        const memberships = await api.tenantMembershipsList();

        const foundTenant = memberships.data.rows?.find(
          (m) => m.tenant?.metadata.id === tenantId,
        )?.tenant;

        switch (foundTenant?.version) {
          case TenantVersion.V0:
            navigate(`/workflow-runs?tenant=${tenantId}`);
            break;
          case TenantVersion.V1:
            navigate(`/v1/runs?tenant=${tenantId}`);
            break;
          default:
            navigate('/');
            break;
        }
      } catch (e) {
        navigate('/');
      }
    },
    onError: handleApiError,
  });

  const rejectMutation = useMutation({
    mutationKey: ['tenant-invite:reject'],
    mutationFn: async (data: { invite: string }) => {
      await api.tenantInviteReject(data);
    },
    onSuccess: async () => {
      navigate('/');
    },
    onError: handleApiError,
  });

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
                  <p className="text-sm text-gray-700 dark:text-gray-300 mb-4">
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
                          tenantId: invite.tenantId,
                          inner: {
                            invite: invite.metadata.id,
                          },
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
