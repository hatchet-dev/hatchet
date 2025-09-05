import { useMemo, useCallback } from 'react';
import { useQuery, useMutation } from '@tanstack/react-query';
import { cloudApi } from '@/lib/api/api';
import useCloudApiMeta from '@/pages/auth/hooks/use-cloud-api-meta';
import { useApiError } from '@/lib/hooks';
import {
  CreateManagementTokenResponse,
  ManagementTokenDuration,
  OrganizationMember,
} from '@/lib/api/generated/cloud/data-contracts';

export function useOrganizations() {
  const { isCloudEnabled } = useCloudApiMeta();
  const { handleApiError } = useApiError({});

  const organizationListQuery = useQuery({
    queryKey: ['organization:list'],
    queryFn: async () => {
      const result = await cloudApi.organizationList();
      return result.data;
    },
    enabled: isCloudEnabled,
  });

  const organizations = useMemo(
    () => organizationListQuery.data?.rows || [],
    [organizationListQuery.data?.rows],
  );

  const getOrganizationForTenant = useCallback(
    (tenantId: string) => {
      return organizations.find((org) => org.tenants.includes(tenantId));
    },
    [organizations],
  );

  const getOrganizationIdForTenant = useCallback(
    (tenantId: string) => {
      const org = getOrganizationForTenant(tenantId);
      return org?.metadata.id || null;
    },
    [getOrganizationForTenant],
  );

  const hasOrganizations = useMemo(() => {
    return (organizationListQuery.data?.rows?.length || 0) > 0;
  }, [organizationListQuery.data?.rows]);

  const acceptOrgInviteMutation = useMutation({
    mutationKey: ['organization-invite:accept'],
    mutationFn: async (data: { inviteId: string }) => {
      await cloudApi.organizationInviteAccept({
        id: data.inviteId,
      });
    },
    onError: handleApiError,
  });

  const rejectOrgInviteMutation = useMutation({
    mutationKey: ['organization-invite:reject'],
    mutationFn: async (data: { inviteId: string }) => {
      await cloudApi.organizationInviteReject({
        id: data.inviteId,
      });
    },
    onError: handleApiError,
  });

  const createTenantMutation = useMutation({
    mutationFn: async (data: {
      organizationId: string;
      name: string;
      slug: string;
    }) => {
      const result = await cloudApi.organizationCreateTenant(
        data.organizationId,
        {
          name: data.name,
          slug: data.slug,
        },
      );
      return result.data;
    },
    onError: handleApiError,
  });

  const cancelInviteMutation = useMutation({
    mutationFn: async (data: { inviteId: string }) => {
      await cloudApi.organizationInviteDelete(data.inviteId);
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
      duration: ManagementTokenDuration;
    }) => {
      const result = await cloudApi.managementTokenCreate(data.organizationId, {
        name: data.name,
        duration: data.duration,
      });
      return result.data;
    },
    onError: handleApiError,
  });

  const deleteMemberMutation = useMutation({
    mutationFn: async (data: { memberId: string; email: string }) => {
      await cloudApi.organizationMemberDelete(data.memberId, {
        emails: [data.email],
      });
    },
    onError: handleApiError,
  });

  const deleteTokenMutation = useMutation({
    mutationFn: async (data: { tokenId: string }) => {
      await cloudApi.managementTokenDelete(data.tokenId);
    },
    onError: handleApiError,
  });

  const handleCreateToken = useCallback(
    (
      organizationId: string,
      name: string,
      duration: ManagementTokenDuration,
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

  return {
    organizations,
    organizationData: organizationListQuery.data,
    isCloudEnabled,
    getOrganizationForTenant,
    getOrganizationIdForTenant,
    hasOrganizations,
    acceptOrgInviteMutation,
    rejectOrgInviteMutation,
    createTenantMutation,
    handleCancelInvite,
    handleCreateToken,
    handleDeleteMember,
    handleDeleteToken,
    // Loading states for mutations
    cancelInviteLoading: cancelInviteMutation.isPending,
    createTokenLoading: createTokenMutation.isPending,
    deleteMemberLoading: deleteMemberMutation.isPending,
    deleteTokenLoading: deleteTokenMutation.isPending,
    isLoading: organizationListQuery.isLoading,
    error: organizationListQuery.error,
  };
}
