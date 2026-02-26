import { Button } from '@/components/v1/ui/button';
import { useAnalytics } from '@/hooks/use-analytics';
import { useOrganizations } from '@/hooks/use-organizations';
import { useTenantDetails } from '@/hooks/use-tenant';
import api from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { appRoutes } from '@/router';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { redirect, useLoaderData, useNavigate } from '@tanstack/react-router';
import { useEffect } from 'react';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function loader(_args: { request: Request }) {
  // Avoid calling cloud-only endpoints (like /management/invites) unless cloud is enabled.
  // In OSS environments, cloud endpoints can return a 403 and create noisy console logs.
  let isCloudEnabled = false;

  try {
    const meta = await cloudApi.metadataGet();
    // In OSS, the API returns an `errors` field instead of cloud metadata.
    // @ts-expect-error `errors` may be present in OSS mode
    isCloudEnabled = !!meta?.data && !meta?.data?.errors;
  } catch {
    isCloudEnabled = false;
  }

  const [tenantInvitesRes, orgInvitesRes] = await Promise.allSettled([
    api.userListTenantInvites(),
    isCloudEnabled
      ? cloudApi
          .userListOrganizationInvites()
          .catch(() => ({ data: { rows: [] } }))
      : Promise.resolve({ data: { rows: [] } }),
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
  const queryClient = useQueryClient();
  const { handleApiError } = useApiError({});
  const { setTenant } = useTenantDetails();
  const { acceptOrgInviteMutation, rejectOrgInviteMutation } =
    useOrganizations();
  const { capture } = useAnalytics();
  const { invalidate: invalidateUserUniverse, get: getUserUniverse } =
    useUserUniverse();

  const { tenantInvites, orgInvites } = useLoaderData({
    from: appRoutes.onboardingInvitesRoute.to,
  }) as Awaited<ReturnType<typeof loader>>;

  // Track invites page view
  useEffect(() => {
    capture('onboarding_invites_viewed', {
      tenant_invites_count: tenantInvites.length,
      org_invites_count: orgInvites.length,
      total_invites: tenantInvites.length + orgInvites.length,
    });
  }, [capture, tenantInvites.length, orgInvites.length]);

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
      await queryClient.invalidateQueries({
        queryKey: ['pending-invites'],
      });

      invalidateUserUniverse();

      const { tenantMemberships } = await getUserUniverse();

      const membership = tenantMemberships.find(
        (m) => m.tenant?.metadata.id === tenantId,
      );

      if (membership?.tenant) {
        setTenant(membership.tenant);
        capture('onboarding_tenant_invite_accepted', {
          tenant_id: tenantId,
        });
        navigate({
          to: appRoutes.tenantOverviewRoute.to,
          params: { tenant: tenantId },
        });
      } else {
        throw new Error('Tenant not found after accepting invite');
      }
    },
    onError: handleApiError,
  });

  const rejectMutation = useMutation({
    mutationKey: ['tenant-invite:reject'],
    mutationFn: async (data: { invite: string }) => {
      await api.tenantInviteReject(data);
      return data.invite;
    },
    onSuccess: async (inviteId: string) => {
      await queryClient.invalidateQueries({
        queryKey: ['pending-invites'],
      });
      capture('onboarding_tenant_invite_rejected', {
        invite_id: inviteId,
      });
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
                            onSuccess: async () => {
                              await queryClient.invalidateQueries({
                                queryKey: ['pending-invites'],
                              });
                              capture('onboarding_org_invite_rejected', {
                                invite_id: invite.metadata.id,
                              });
                              navigate({ to: appRoutes.authenticatedRoute.to });
                            },
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
                            onSuccess: async () => {
                              await queryClient.invalidateQueries({
                                queryKey: ['pending-invites'],
                              });
                              capture('onboarding_org_invite_accepted', {
                                invite_id: invite.metadata.id,
                              });
                              navigate({ to: appRoutes.authenticatedRoute.to });
                            },
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
