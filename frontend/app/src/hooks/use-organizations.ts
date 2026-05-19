import {
  CreateManagementTokenResponse,
  ManagementTokenDuration,
  OrganizationForUser,
  OrganizationMember,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import { useApiError } from '@/lib/hooks';
import { useUserUniverse } from '@/providers/user-universe';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useMemo, useCallback } from 'react';
import invariant from 'tiny-invariant';

/**
 * Hook for organization data and operations
 *
 * Now backed by AppContext for better performance.
 * Gets organization data from context, but keeps all mutation logic here.
 */
export function useOrganizations() {
  const {
    organizations: organizationData,
    isLoaded: isUserUniverseLoaded,
    isCloudEnabled,
  } = useUserUniverse();
  const { handleApiError } = useApiError({});
  const orgApi = useOrganizationApi();

  // Re-query for mutations (will revalidate the context)
  const organizationListQuery = useQuery({
    ...orgApi.organizationListQuery(),
    enabled: isCloudEnabled,
  });

  const organizations = useMemo(() => {
    if (isUserUniverseLoaded && isCloudEnabled) {
      invariant(organizationData);
      return organizationData;
    }
    return [];
  }, [isUserUniverseLoaded, organizationData, isCloudEnabled]);

  const getOrganizationForTenant = useMemo(() => {
    const tenantIdToOrganization = new Map<string, OrganizationForUser>();
    organizations.forEach((org) => {
      org.tenants.forEach((tenant) => {
        tenantIdToOrganization.set(tenant.id, org);
      });
    });
    return (tenantId: string) => tenantIdToOrganization.get(tenantId);
  }, [organizations]);

  const getOrganizationIdForTenant = useCallback(
    (tenantId: string) => {
      const org = getOrganizationForTenant(tenantId);
      return org?.metadata.id || null;
    },
    [getOrganizationForTenant],
  );

  const isTenantArchivedInOrg = useCallback(
    (tenantId: string) => {
      const orgForTenant = getOrganizationForTenant(tenantId);
      if (!orgForTenant) {
        return false; // Not part of any org, so not archived
      }

      return (
        orgForTenant.tenants?.find((tenant) => tenant.id === tenantId)
          ?.status === TenantStatusType.ARCHIVED
      );
    },
    [getOrganizationForTenant],
  );

  const hasOrganizations = useMemo(() => {
    return (organizationListQuery.data?.rows?.length || 0) > 0;
  }, [organizationListQuery.data?.rows]);

  const orgInviteAccept = orgApi.organizationInviteAcceptMutation();
  const acceptOrgInviteMutation = useMutation({
    mutationKey: orgInviteAccept.mutationKey,
    mutationFn: async (data: { inviteId: string }) => {
      await orgInviteAccept.mutationFn({ id: data.inviteId });
    },
    onError: handleApiError,
  });

  const orgInviteReject = orgApi.organizationInviteRejectMutation();
  const rejectOrgInviteMutation = useMutation({
    mutationKey: orgInviteReject.mutationKey,
    mutationFn: async (data: { inviteId: string }) => {
      await orgInviteReject.mutationFn({ id: data.inviteId });
    },
    onError: handleApiError,
  });

  const createTenantMutation = useMutation({
    mutationFn: async (data: {
      organizationId: string;
      name: string;
      slug: string;
    }) => {
      return orgApi
        .organizationCreateTenantMutation(data.organizationId)
        .mutationFn({ name: data.name, slug: data.slug });
    },
    onSuccess: () => {
      localStorage.setItem('hatchet:show-welcome', '1');
    },
    onError: handleApiError,
  });

  const cancelInviteMutation = useMutation({
    mutationFn: async (data: { inviteId: string }) => {
      await orgApi.organizationInviteDeleteMutation(data.inviteId).mutationFn();
    },
    onError: handleApiError,
  });

  const handleCancelInvite = useCallback(
    (
      inviteId: string,
      onSuccess: () => void,
      onOpenChange: (open: boolean) => void,
    ) => {
      cancelInviteMutation.mutate(
        { inviteId: inviteId },
        {
          onSuccess: () => {
            onSuccess();
            onOpenChange(false);
          },
          onError: () => {
            onOpenChange(false);
          },
        },
      );
    },
    [cancelInviteMutation],
  );

  const createTokenMutation = useMutation({
    mutationFn: async (data: {
      organizationId: string;
      name: string;
      duration?: ManagementTokenDuration;
    }) => {
      const body: { name: string; duration?: ManagementTokenDuration } = {
        name: data.name,
      };
      if (data.duration != null) {
        body.duration = data.duration;
      }
      return orgApi
        .managementTokenCreateMutation(data.organizationId)
        .mutationFn(body);
    },
    onError: handleApiError,
  });

  const deleteMemberMutation = useMutation({
    mutationFn: async (data: { memberId: string; email: string }) => {
      await orgApi
        .organizationMemberDeleteMutation(data.memberId)
        .mutationFn({ emails: [data.email] });
    },
    onError: handleApiError,
  });

  const deleteTokenMutation = useMutation({
    mutationFn: async (data: { tokenId: string }) => {
      await orgApi.managementTokenDeleteMutation(data.tokenId).mutationFn();
    },
    onError: handleApiError,
  });

  const deleteTenantMutation = useMutation({
    mutationFn: async (data: { tenantId: string }) => {
      await orgApi.organizationTenantDeleteMutation(data.tenantId).mutationFn();
    },
    onError: handleApiError,
  });

  const updateOrganizationMutation = useMutation({
    mutationFn: async (data: {
      organizationId: string;
      name?: string;
      inactivity_timeout?: string;
    }) => {
      return orgApi.organizationUpdateMutation(data.organizationId).mutationFn({
        name: data.name,
        inactivity_timeout: data.inactivity_timeout,
      });
    },
    onError: handleApiError,
  });

  const orgCreate = orgApi.organizationCreateMutation();
  const createOrganizationMutation = useMutation({
    ...orgCreate,
    onError: handleApiError,
    onSuccess: () => {
      organizationListQuery.refetch();
    },
  });

  const createOrganizationSsoDomainMutation = useMutation({
    mutationFn: async (data: { organizationId: string; ssoDomain: string }) => {
      return orgApi
        .organizationSsoDomainCreateMutation(data.organizationId)
        .mutationFn(data.ssoDomain);
    },
    onError: handleApiError,
  });

  const deleteOrganizationSsoDomainMutation = useMutation({
    mutationFn: async (data: { organizationId: string; ssoDomain: string }) => {
      return orgApi
        .organizationSsoDomainDeleteMutation(data.organizationId)
        .mutationFn(data.ssoDomain);
    },
    onError: handleApiError,
  });

  const handleCreateToken = useCallback(
    (
      organizationId: string,
      name: string,
      duration: ManagementTokenDuration | undefined,
      onSuccess: (data: CreateManagementTokenResponse) => void,
    ) => {
      createTokenMutation.mutate(
        { organizationId, name, duration },
        {
          onSuccess: (data) => {
            onSuccess(data);
          },
          onError: () => {
            // Error handling is done by the mutation itself via handleApiError
          },
        },
      );
    },
    [createTokenMutation],
  );

  const handleDeleteMember = useCallback(
    (
      member: OrganizationMember,
      onSuccess: () => void,
      onOpenChange: (open: boolean) => void,
    ) => {
      if (member) {
        deleteMemberMutation.mutate(
          { memberId: member.metadata.id, email: member.email },
          {
            onSuccess: () => {
              onSuccess();
              onOpenChange(false);
            },
            onError: () => {
              onOpenChange(false);
            },
          },
        );
      }
    },
    [deleteMemberMutation],
  );

  const handleDeleteToken = useCallback(
    (
      tokenId: string,
      onSuccess: () => void,
      onOpenChange: (open: boolean) => void,
    ) => {
      deleteTokenMutation.mutate(
        { tokenId: tokenId },
        {
          onSuccess: () => {
            onSuccess();
            onOpenChange(false);
          },
          onError: () => {
            onOpenChange(false);
          },
        },
      );
    },
    [deleteTokenMutation],
  );

  const handleDeleteTenant = useCallback(
    (
      tenantId: string,
      onSuccess: () => void,
      onOpenChange: (open: boolean) => void,
    ) => {
      deleteTenantMutation.mutate(
        { tenantId: tenantId },
        {
          onSuccess: () => {
            onSuccess();
            onOpenChange(false);
          },
          onError: () => {
            onOpenChange(false);
          },
        },
      );
    },
    [deleteTenantMutation],
  );

  const handleUpdateOrganization = useCallback(
    (organizationId: string, name: string, onSuccess: () => void) => {
      updateOrganizationMutation.mutate(
        { organizationId, name },
        {
          onSuccess: () => {
            onSuccess();
          },
          onError: () => {
            // Error handling is done by the mutation itself via handleApiError
          },
        },
      );
    },
    [updateOrganizationMutation],
  );

  const handleUpdateOrganizationTimeout = (
    organizationId: string,
    inactivityTimeoutMs: number,
    onSuccess: () => void,
  ) => {
    updateOrganizationMutation.mutate(
      { organizationId, inactivity_timeout: `${inactivityTimeoutMs}ms` },
      {
        onSuccess: () => {
          onSuccess();
        },
        onError: () => {},
      },
    );
  };

  const handleCreateOrganization = useCallback(
    (name: string, onSuccess: (organizationId: string) => void) => {
      createOrganizationMutation.mutate(
        { name },
        {
          onSuccess: (data) => {
            onSuccess(data.metadata.id);
          },
          onError: () => {
            // Error handling is done by the mutation itself via handleApiError
          },
        },
      );
    },
    [createOrganizationMutation],
  );

  const handleCreateOrganizationSsoDomain = useCallback(
    (
      organizationId: string,
      ssoDomain: string,
      onSuccess: (organizationId: string) => void,
      onError?: () => void,
    ) => {
      createOrganizationSsoDomainMutation.mutate(
        { organizationId, ssoDomain },
        {
          onSuccess: () => {
            onSuccess(organizationId);
          },
          onError: () => {
            onError?.();
          },
        },
      );
    },
    [createOrganizationSsoDomainMutation],
  );

  const handleDeleteOrganizationSsoDomain = useCallback(
    (
      organizationId: string,
      ssoDomain: string,
      onSuccess: (organizationId: string) => void,
    ) => {
      deleteOrganizationSsoDomainMutation.mutate(
        { organizationId, ssoDomain },
        {
          onSuccess: () => {
            onSuccess(organizationId);
          },
          onError: () => {
            // Error handling is done by the mutation itself via handleApiError
          },
        },
      );
    },
    [deleteOrganizationSsoDomainMutation],
  );

  return {
    organizations,
    organizationData, // From context
    isCloudEnabled,
    getOrganizationForTenant,
    getOrganizationIdForTenant,
    isTenantArchivedInOrg,
    hasOrganizations,
    acceptOrgInviteMutation,
    rejectOrgInviteMutation,
    createTenantMutation,
    handleCancelInvite,
    handleCreateToken,
    handleDeleteMember,
    handleDeleteToken,
    handleDeleteTenant,
    handleUpdateOrganization,
    handleUpdateOrganizationTimeout,
    handleCreateOrganization,
    handleCreateOrganizationSsoDomain,
    handleDeleteOrganizationSsoDomain,
    // Loading states for mutations
    cancelInviteLoading: cancelInviteMutation.isPending,
    createTokenLoading: createTokenMutation.isPending,
    deleteMemberLoading: deleteMemberMutation.isPending,
    deleteTokenLoading: deleteTokenMutation.isPending,
    deleteTenantLoading: deleteTenantMutation.isPending,
    updateOrganizationLoading: updateOrganizationMutation.isPending,
    createOrganizationLoading: createOrganizationMutation.isPending,
    isUserUniverseLoaded,
    error: organizationListQuery.error,
  };
}
