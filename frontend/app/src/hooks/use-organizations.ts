import useCloud from '@/hooks/use-cloud';
import useControlPlane from '@/hooks/use-control-plane';
import { useOrganizationApi } from '@/lib/api/organization-wrapper';
import {
  CreateManagementTokenResponse,
  ManagementTokenDuration,
  OrganizationMember,
  TenantStatusType,
} from '@/lib/api/generated/cloud/data-contracts';
import { useApiError } from '@/lib/hooks';
import { useAppContext } from '@/providers/app-context';
import { useQuery, useMutation } from '@tanstack/react-query';
import { useMemo, useCallback } from 'react';

/**
 * Hook for organization data and operations
 *
 * Now backed by AppContext for better performance.
 * Gets organization data from context, but keeps all mutation logic here.
 */
export function useOrganizations() {
  const {
    organizations: organizationData,
    isCloudEnabled,
    isControlPlaneEnabled,
  } = useAppContext();
  const { handleApiError } = useApiError({});
  const orgApi = useOrganizationApi();
  const { isCloudLoading } = useCloud();
  const { isControlPlaneLoading } = useControlPlane();

  // Re-query for mutations (will revalidate the context)
  const organizationListQuery = useQuery({
    queryKey: ['organization:list'],
    queryFn: async () => {
      const result = await orgApi.organizationList();
      return result.data;
    },
    enabled:
      (isCloudEnabled || isControlPlaneEnabled) &&
      !isCloudLoading &&
      !isControlPlaneLoading,
  });

  const organizations = useMemo(
    () => organizationData?.rows || [],
    [organizationData?.rows],
  );

  const getOrganizationForTenant = useCallback(
    (tenantId: string) => {
      return organizations.find((org) =>
        (org.tenants || []).some((tenant) => tenant.id === tenantId),
      );
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
    return organizations.length > 0;
  }, [organizations]);

  const acceptOrgInviteMutation = useMutation({
    mutationKey: ['organization-invite:accept'],
    mutationFn: async (data: { inviteId: string }) => {
      await orgApi.organizationInviteAccept({
        id: data.inviteId,
      });
    },
    onError: handleApiError,
  });

  const rejectOrgInviteMutation = useMutation({
    mutationKey: ['organization-invite:reject'],
    mutationFn: async (data: { inviteId: string }) => {
      await orgApi.organizationInviteReject({
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
      const result = await orgApi.organizationCreateTenant(
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
      await orgApi.organizationInviteDelete(data.inviteId);
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
      const result = await orgApi.managementTokenCreate(
        data.organizationId,
        body,
      );
      return result.data;
    },
    onError: handleApiError,
  });

  const deleteMemberMutation = useMutation({
    mutationFn: async (data: { memberId: string; email: string }) => {
      await orgApi.organizationMemberDelete(data.memberId, {
        emails: [data.email],
      });
    },
    onError: handleApiError,
  });

  const deleteTokenMutation = useMutation({
    mutationFn: async (data: { tokenId: string }) => {
      await orgApi.managementTokenDelete(data.tokenId);
    },
    onError: handleApiError,
  });

  const deleteTenantMutation = useMutation({
    mutationFn: async (data: { tenantId: string }) => {
      await orgApi.organizationTenantDelete(data.tenantId);
    },
    onError: handleApiError,
  });

  const updateOrganizationMutation = useMutation({
    mutationFn: async (data: { organizationId: string; name: string }) => {
      const result = await orgApi.organizationUpdate(data.organizationId, {
        name: data.name,
      });
      return result.data;
    },
    onError: handleApiError,
  });

  const createOrganizationMutation = useMutation({
    mutationFn: async (data: { name: string }) => {
      const result = await orgApi.organizationCreate({
        name: data.name,
      });
      return result.data;
    },
    onError: handleApiError,
    onSuccess: () => {
      organizationListQuery.refetch();
    },
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

  return {
    organizations,
    organizationData, // From context
    isCloudEnabled,
    isControlPlaneEnabled,
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
    handleCreateOrganization,
    // Loading states for mutations
    cancelInviteLoading: cancelInviteMutation.isPending,
    createTokenLoading: createTokenMutation.isPending,
    deleteMemberLoading: deleteMemberMutation.isPending,
    deleteTokenLoading: deleteTokenMutation.isPending,
    deleteTenantLoading: deleteTenantMutation.isPending,
    updateOrganizationLoading: updateOrganizationMutation.isPending,
    createOrganizationLoading: createOrganizationMutation.isPending,
    isLoading: organizationListQuery.isLoading,
    error: organizationListQuery.error,
  };
}
