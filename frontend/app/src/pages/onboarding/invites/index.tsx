import { Button } from '@/components/v1/ui/button';
import { useAnalytics } from '@/hooks/use-analytics';
import { useOrganizations } from '@/hooks/use-organizations';
import {
  pendingInvitesQuery,
  usePendingInvites,
} from '@/hooks/use-pending-invites';
import { useTenantDetails } from '@/hooks/use-tenant';
import api, { Tenant, TenantInvite } from '@/lib/api';
import { cloudApi } from '@/lib/api/api';
import type { OrganizationInvite } from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { metadataIndicatesCloudEnabled } from '@/pages/auth/hooks/use-cloud';
import { useUserUniverse } from '@/providers/user-universe';
import queryClient from '@/query-client';
import { appRoutes } from '@/router';
import { useMutation } from '@tanstack/react-query';
import { redirect, useLoaderData, useNavigate } from '@tanstack/react-router';
import { useCallback, useEffect, useState } from 'react';
import invariant from 'tiny-invariant';

// eslint-disable-next-line @typescript-eslint/no-unused-vars
export async function loader(_args: { request: Request }) {
  // Avoid calling cloud-only endpoints (like /management/invites) unless cloud is enabled.
  // In OSS environments, cloud endpoints can return a 403 and create noisy console logs.
  let isCloudEnabled = false;

  try {
    const { data: meta } = await cloudApi.metadataGet();
    isCloudEnabled = metadataIndicatesCloudEnabled(meta);
  } catch {
    isCloudEnabled = false;
  }

  const { tenantInvites, organizationInvites, inviteCount } =
    await queryClient.fetchQuery(pendingInvitesQuery(isCloudEnabled));

  // Doesn't work right now because you don't have any access to organizations you're not a member of
  // const organizationInvitesWithOrganizations = await Promise.all(
  //   orgInvites.map(async (invite) => ({
  //     ...invite,
  //     organization: (await cloudApi.organizationGet(invite.organizationId))
  //       .data,
  //   })),
  // );

  if (inviteCount === 0) {
    throw redirect({ to: appRoutes.authenticatedRoute.to });
  }

  return {
    tenantInvites,
    organizationInvites,
  };
}

const OrganizationInviteList = ({
  orgInvites,
  onDealtWithInvite,
}: {
  orgInvites: OrganizationInvite[];
  onDealtWithInvite: (organizationId: string, accepted: boolean) => void;
}) => {
  const { acceptOrgInviteMutation, rejectOrgInviteMutation } =
    useOrganizations();
  const { capture } = useAnalytics();
  const { invalidate: invalidateUserUniverse, get: getUserUniverse } =
    useUserUniverse();

  return (
    <>
      {orgInvites.map((invite) => {
        return (
          <div
            key={invite.metadata.id}
            className="flex flex-col space-y-2 text-center"
          >
            <p className="mb-4 text-sm text-gray-700 dark:text-gray-300">
              You have been invited to join an organization by{' '}
              <strong>{invite.inviterEmail}</strong> on Hatchet.
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
                        capture('onboarding_org_invite_rejected', {
                          invite_id: invite.metadata.id,
                        });
                        onDealtWithInvite(invite.organizationId, false);
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
                        capture('onboarding_org_invite_accepted', {
                          invite_id: invite.metadata.id,
                        });
                        invalidateUserUniverse();
                        const refetchingUserUniversePromise = getUserUniverse();
                        onDealtWithInvite(invite.organizationId, true);
                        return refetchingUserUniversePromise;
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
    </>
  );
};

const TenantInviteList = ({
  tenantInvites,
  onDealtWithInvite,
}: {
  tenantInvites: TenantInvite[];
  onDealtWithInvite: (tenantId: string, accepted: boolean) => void;
}) => {
  const { capture } = useAnalytics();
  const { handleApiError } = useApiError({});
  const { invalidate: invalidateUserUniverse, get: getUserUniverse } =
    useUserUniverse();

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
      invalidateUserUniverse();

      const { tenantMemberships } = await getUserUniverse();

      const membership = tenantMemberships.find(
        (m) => m.tenant?.metadata.id === tenantId,
      );

      if (membership?.tenant) {
        capture('onboarding_tenant_invite_accepted', {
          tenant_id: tenantId,
        });
        onDealtWithInvite(tenantId, true);
      } else {
        throw new Error('Tenant not found after accepting invite');
      }
    },
    onError: handleApiError,
  });

  const rejectMutation = useMutation({
    mutationKey: ['tenant-invite:reject'],
    mutationFn: async (data: { invite: string; tenantId: string }) => {
      await api.tenantInviteReject(data);
      return { inviteId: data.invite, tenantId: data.tenantId };
    },
    onSuccess: async ({
      inviteId,
      tenantId,
    }: {
      inviteId: string;
      tenantId: string;
    }) => {
      capture('onboarding_tenant_invite_rejected', {
        invite_id: inviteId,
      });
      onDealtWithInvite(tenantId, false);
    },
    onError: handleApiError,
  });

  return (
    <>
      {tenantInvites.map((invite) => {
        return (
          <div
            key={invite.metadata.id}
            className="flex flex-col space-y-2 text-center"
          >
            <p className="mb-4 text-sm text-gray-700 dark:text-gray-300">
              You have been invited to join the{' '}
              <strong>{invite.tenantName}</strong> tenant by{' '}
              <strong>{invite.email}</strong> on Hatchet.
            </p>
            <div className="flex flex-row justify-center gap-2">
              <Button
                variant="outline"
                className="w-full"
                onClick={() => {
                  rejectMutation.mutate({
                    invite: invite.metadata.id,
                    tenantId: invite.tenantId,
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
    </>
  );
};

export default function Invites() {
  const { capture } = useAnalytics();
  const navigate = useNavigate();
  const [lastAcceptedInvite, setLastAcceptedInvite] = useState<
    | { type: 'tenant'; tenantId: string }
    | { type: 'organization'; organizationId: string }
    | null
  >(null);
  const { isCloudEnabled, tenantMemberships, organizations } =
    useUserUniverse();
  const { invalidate: invalidatePendingInvites } = usePendingInvites();
  const { setTenant } = useTenantDetails();

  const {
    tenantInvites: initialTenantInvites,
    organizationInvites: initialOrgInvites,
  } = useLoaderData({
    from: appRoutes.onboardingInvitesRoute.to,
  }) as Awaited<ReturnType<typeof loader>>;

  const [tenantInvites, setTenantInvites] = useState(initialTenantInvites);
  const [orgInvites, setOrgInvites] = useState(initialOrgInvites);

  const [
    tenantAssociatedWithLastAcceptedInvite,
    setTenantAssociatedWithLastAcceptedInvite,
  ] = useState<Tenant | null>(null);

  const totalInvites = tenantInvites.length + orgInvites.length;

  const getTenant = useCallback(
    (tenantId: string) => {
      if (!tenantMemberships) {
        return null;
      }

      const membership = tenantMemberships.find(
        (m) => m.tenant?.metadata.id === tenantId,
      );
      invariant(membership?.tenant);
      return membership.tenant;
    },
    [tenantMemberships],
  );

  useEffect(() => {
    if (!lastAcceptedInvite) {
      return;
    }

    if (lastAcceptedInvite.type === 'tenant') {
      const tenant = getTenant(lastAcceptedInvite.tenantId);
      if (tenant) {
        setTenantAssociatedWithLastAcceptedInvite(tenant);
      }
    } else if (isCloudEnabled && organizations) {
      const organization = organizations.find(
        (org) => org.metadata.id === lastAcceptedInvite.organizationId,
      );
      if (organization) {
        const tenant = getTenant(organization.tenants?.[0]?.id);
        if (tenant) {
          setTenantAssociatedWithLastAcceptedInvite(tenant);
        }
      }
    }
  }, [lastAcceptedInvite, getTenant, isCloudEnabled, organizations]);

  const navigateIfAppropriate = useCallback(async () => {
    if (totalInvites > 0) {
      return;
    }

    invalidatePendingInvites();

    if (!lastAcceptedInvite && !tenantAssociatedWithLastAcceptedInvite) {
      navigate({
        to: appRoutes.authenticatedRoute.to,
      });
      return;
    }

    if (tenantAssociatedWithLastAcceptedInvite) {
      // IMPLICIT NAVIGATION TO THE TENANT PAGE
      setTenant(tenantAssociatedWithLastAcceptedInvite);
    }
  }, [
    lastAcceptedInvite,
    navigate,
    totalInvites,
    invalidatePendingInvites,
    setTenant,
    tenantAssociatedWithLastAcceptedInvite,
  ]);

  useEffect(() => {
    capture('onboarding_invites_viewed', {
      tenant_invites_count: tenantInvites.length,
      org_invites_count: orgInvites.length,
      total_invites: tenantInvites.length + orgInvites.length,
    });
  }, [capture, tenantInvites.length, orgInvites.length]);

  useEffect(() => {
    navigateIfAppropriate();
  }, [navigateIfAppropriate]);

  if (totalInvites === 0) {
    return <></>;
  }

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
            <OrganizationInviteList
              orgInvites={orgInvites}
              onDealtWithInvite={(organizationId, accepted) => {
                if (accepted) {
                  setLastAcceptedInvite({
                    type: 'organization',
                    organizationId,
                  });
                }

                setOrgInvites((prevOrgInvites) =>
                  prevOrgInvites.filter(
                    (invite) => invite.organizationId !== organizationId,
                  ),
                );
              }}
            />
            <TenantInviteList
              tenantInvites={tenantInvites}
              onDealtWithInvite={(tenantId, accepted) => {
                if (accepted) {
                  setLastAcceptedInvite({ type: 'tenant', tenantId });
                }

                setTenantInvites((prevTenantInvites) =>
                  prevTenantInvites.filter(
                    (invite) => invite.tenantId !== tenantId,
                  ),
                );
              }}
            />
          </div>
        </div>
      </div>
    </div>
  );
}
