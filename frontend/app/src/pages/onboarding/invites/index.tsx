import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { useMutation } from '@tanstack/react-query';
import { redirect, useLoaderData, useNavigate } from '@tanstack/react-router';
import { Button } from '@/components/v1/ui/button';
import { useOrganizations } from '@/hooks/use-organizations';
import { appRoutes } from '@/router';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function loader(_args: { request: Request }) {
  const [tenantInvitesRes, orgInvitesRes] = await Promise.allSettled([
    api.userListTenantInvites(),
    cloudApi
      .userListOrganizationInvites()
      .catch(() => ({ data: { rows: [] } })),
  ]);

  const tenantInvites =
    tenantInvitesRes.status === 'fulfilled'
      ? tenantInvitesRes.value.data.rows || []
      : [];
  const orgInvites =
    orgInvitesRes.status === 'fulfilled'
      ? orgInvitesRes.value.data.rows || []
      : [];

  if (tenantInvites.length === 0 && orgInvites.length === 0) {
    throw redirect({ to: appRoutes.authenticatedRoute.to });
  }

  return {
    tenantInvites,
    orgInvites,
  };
}

export default function Invites() {
  const navigate = useNavigate();
  const { handleApiError } = useApiError({});
  const { acceptOrgInviteMutation, rejectOrgInviteMutation } =
    useOrganizations();

  const { tenantInvites, orgInvites } = useLoaderData({
    from: appRoutes.onboardingInvitesRoute.to,
  }) as Awaited<ReturnType<typeof loader>>;

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
      navigate({
        to: appRoutes.tenantRunsRoute.to,
        params: { tenant: tenantId },
      });
    },
    onError: handleApiError,
  });

  const rejectMutation = useMutation({
    mutationKey: ['tenant-invite:reject'],
    mutationFn: async (data: { invite: string }) => {
      await api.tenantInviteReject(data);
    },
    onSuccess: async () => {
      navigate({ to: appRoutes.authenticatedRoute.to });
    },
    onError: handleApiError,
  });

  const totalInvites = tenantInvites.length + orgInvites.length;
  const header =
    totalInvites > 1
      ? 'Join your teams'
      : tenantInvites.length > 0
        ? 'Join ' + tenantInvites[0].tenantName
        : 'Join ' + orgInvites[0].inviterEmail + "'s organization";

  return (
    <div className="flex h-full w-full flex-1 flex-row">
      <div className="container relative hidden flex-col items-center justify-center md:grid lg:max-w-none lg:grid-cols-2 lg:px-0">
        <div className="mx-auto w-screen lg:p-8">
          <div className="mx-auto flex w-40 flex-col justify-center space-y-6 sm:w-[350px]">
            <div className="flex flex-col space-y-2 text-center">
              <h1 className="text-2xl font-semibold tracking-tight">
                {header}
              </h1>
            </div>
            {tenantInvites.map((invite) => {
              return (
                <div
                  key={invite.metadata.id}
                  className="flex flex-col space-y-2 text-center"
                >
                  <p className="mb-4 text-sm text-gray-700 dark:text-gray-300">
                    You got an invitation to join {invite.tenantName} on
                    Hatchet.
                  </p>
                  <div className="flex flex-row justify-center gap-2">
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
            {orgInvites.map((invite) => {
              return (
                <div
                  key={invite.metadata.id}
                  className="flex flex-col space-y-2 text-center"
                >
                  <p className="mb-4 text-sm text-gray-700 dark:text-gray-300">
                    You got an invitation to join an organization from{' '}
                    {invite.inviterEmail} on Hatchet.
                  </p>
                  <div className="flex flex-row justify-center gap-2">
                    <Button
                      variant="outline"
                      className="w-full"
                      onClick={() => {
                        rejectOrgInviteMutation.mutate(
                          {
                            inviteId: invite.metadata.id,
                          },
                          {
                            onSuccess: () =>
                              navigate({ to: appRoutes.authenticatedRoute.to }),
                          },
                        );
                      }}
                    >
                      Decline
                    </Button>
                    <Button
                      className="w-full"
                      onClick={() => {
                        acceptOrgInviteMutation.mutate(
                          {
                            inviteId: invite.metadata.id,
                          },
                          {
                            onSuccess: () =>
                              navigate({ to: appRoutes.authenticatedRoute.to }),
                          },
                        );
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
