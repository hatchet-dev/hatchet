import useControlPlane from '@/hooks/use-control-plane';
import { cloudApi, controlPlaneApi } from '@/lib/api/api';
import type { CreateNewTenantForOrganizationRequest as CloudCreateNewTenantForOrganizationRequest } from '@/lib/api/generated/cloud/data-contracts';
import type {
  CreateNewTenantForOrganizationRequest as ControlPlaneCreateNewTenantForOrganizationRequest,
  CreateOrganizationInviteRequest as ControlPlaneCreateOrganizationInviteRequest,
  CreateOrganizationRequest as ControlPlaneCreateOrganizationRequest,
  OrganizationMemberRoleType as ControlPlaneOrganizationMemberRoleType,
  UpdateOrganizationMemberRequest as ControlPlaneUpdateOrganizationMemberRequest,
} from '@/lib/api/generated/control-plane/data-contracts';
import { useMemo } from 'react';

// The cloud request plus the control-plane-only onboarding fields.
// `whatToBuild`/`sdk` must only be sent when the control plane is enabled.
type OrganizationCreateRequest = Parameters<
  typeof cloudApi.organizationCreate
>[0] &
  Pick<ControlPlaneCreateOrganizationRequest, 'whatToBuild' | 'sdk'>;
type OrganizationUpdateRequest = {
  name?: string;
  inactivity_timeout?: string;
};
type OrganizationCreateTenantRequest =
  | CloudCreateNewTenantForOrganizationRequest
  | ControlPlaneCreateNewTenantForOrganizationRequest;
type OrganizationMemberDeleteRequest = Parameters<
  typeof cloudApi.organizationMemberDelete
>[1];
// Control-plane-only: the legacy cloud management API is deprecated and does
// not support member role changes. The role is typed as the shared string
// values so callers holding the cloud client's (nominally distinct) role enum
// compile against it.
type OrganizationMemberUpdateRequest = {
  role: `${ControlPlaneOrganizationMemberRoleType}`;
};
type ManagementTokenCreateRequest = Parameters<
  typeof cloudApi.managementTokenCreate
>[1];
type OrganizationInviteAcceptRequest = Parameters<
  typeof cloudApi.organizationInviteAccept
>[0];
type OrganizationInviteRejectRequest = Parameters<
  typeof cloudApi.organizationInviteReject
>[0];
// The cloud request plus the control-plane-only grant fields.
// `tenants`/`userGroupIds` must only be sent when the control plane is
// enabled.
export type OrganizationInviteCreateRequest = Parameters<
  typeof cloudApi.organizationInviteCreate
>[1] &
  Pick<ControlPlaneCreateOrganizationInviteRequest, 'tenants' | 'userGroupIds'>;
type OrganizationTenantMembersAddRequest = Parameters<
  typeof cloudApi.organizationTenantMembersAdd
>[2] & { canViewPayloads?: boolean };

export function useOrganizationApi() {
  const { isControlPlaneEnabled } = useControlPlane();

  return useMemo(
    () => ({
      // ── Queries ────────────────────────────────────────────────────────────

      organizationListQuery: () => ({
        queryKey: ['organization:list'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationList()
              : cloudApi.organizationList())
          ).data,
      }),

      organizationGetQuery: (organization: string) => ({
        queryKey: ['organization:get', organization] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationGet(organization)
              : cloudApi.organizationGet(organization))
          ).data,
      }),

      organizationAvailableShardsQuery: (organization: string) => ({
        queryKey: ['organization:available-shards', organization] as const,
        queryFn: async () =>
          (await controlPlaneApi.organizationListAvailableShards(organization))
            .data,
      }),

      sharedShardsQuery: () => ({
        queryKey: ['shards:list:shared'] as const,
        queryFn: async () => (await controlPlaneApi.shardsListShared()).data,
      }),

      organizationSsoDomainGetQuery: (organization: string) => ({
        queryKey: ['organization:sso_domain:get', organization] as const,
        queryFn: async () =>
          (await controlPlaneApi.ssoDomainList(organization)).data,
      }),

      organizationSsoConfigGetQuery: (organization: string) => ({
        queryKey: ['organization:sso_config:get', organization] as const,
        queryFn: async () =>
          (await controlPlaneApi.ssoConfigGet(organization)).data,
      }),

      organizationEntitlementsGetQuery: (organization: string) => ({
        queryKey: ['organization:entitlements:get', organization] as const,
        queryFn: async () =>
          (await controlPlaneApi.organizationEntitlementsGet(organization))
            .data,
      }),

      managementTokenListQuery: (organization: string) => ({
        queryKey: ['management-tokens:list', organization] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.managementTokenList(organization)
              : cloudApi.managementTokenList(organization))
          ).data,
      }),

      userListOrganizationInvitesQuery: () => ({
        queryKey: ['user:list:organization-invites'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.userListOrganizationInvites()
              : cloudApi.userListOrganizationInvites())
          ).data,
      }),

      organizationInviteListQuery: (organization: string) => ({
        queryKey: ['organization-invites:list', organization] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationInviteList(organization)
              : cloudApi.organizationInviteList(organization))
          ).data,
      }),

      // ── Mutations ──────────────────────────────────────────────────────────

      organizationSsoConfigUpdateMutation: (organization: string) => ({
        mutationKey: ['organization:sso_config:update', organization] as const,
        mutationFn: async (forceSSO: boolean) => {
          return (
            await controlPlaneApi.ssoConfigUpdate(organization, { forceSSO })
          ).data;
        },
      }),

      organizationSsoDomainCreateMutation: (organization: string) => ({
        mutationKey: ['organization:sso_domain:create', organization] as const,
        mutationFn: async (ssoDomain: string) => {
          return (
            await controlPlaneApi.ssoDomainCreate(organization, {
              ssoDomain: ssoDomain,
            })
          ).data;
        },
      }),

      organizationSsoDomainDeleteMutation: (organization: string) => ({
        mutationKey: ['organization:sso_domain:create', organization] as const,
        mutationFn: async (ssoDomain: string) => {
          return (await controlPlaneApi.ssoDomainDelete(ssoDomain)).data;
        },
      }),

      organizationCreateMutation: () => ({
        mutationKey: ['organization:create'] as const,
        mutationFn: async (data: OrganizationCreateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationCreate(data)
              : // The legacy cloud API does not accept the onboarding
                // fields, so only the name is forwarded.
                cloudApi.organizationCreate({ name: data.name }))
          ).data,
      }),

      organizationUpdateMutation: (organization: string) => ({
        mutationKey: ['organization:update', organization] as const,
        mutationFn: async (data: OrganizationUpdateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationUpdate(organization, data)
              : cloudApi.organizationUpdate(
                  organization,
                  data as { name: string },
                ))
          ).data,
      }),

      organizationCreateTenantMutation: (organization: string) => ({
        mutationKey: ['organization:create-tenant', organization] as const,
        mutationFn: async (data: OrganizationCreateTenantRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationCreateTenant(organization, data)
              : cloudApi.organizationCreateTenant(organization, data))
          ).data,
      }),

      organizationTenantDeleteMutation: (organizationTenant: string) => ({
        mutationKey: [
          'organization-tenant:delete',
          organizationTenant,
        ] as const,
        mutationFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationTenantDelete(organizationTenant)
              : cloudApi.organizationTenantDelete(organizationTenant))
          ).data,
      }),

      organizationMemberDeleteMutation: (organizationMember: string) => ({
        mutationKey: [
          'organization-member:delete',
          organizationMember,
        ] as const,
        mutationFn: async (data: OrganizationMemberDeleteRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationMemberDelete(
                  organizationMember,
                  data,
                )
              : cloudApi.organizationMemberDelete(organizationMember, data))
          ).data,
      }),

      // Control-plane-only: the legacy cloud management API is deprecated and
      // does not support member role changes.
      organizationMemberUpdateMutation: (organizationMember: string) => ({
        mutationKey: [
          'organization-member:update',
          organizationMember,
        ] as const,
        mutationFn: async (data: OrganizationMemberUpdateRequest) =>
          (
            await controlPlaneApi.organizationMemberUpdate(
              organizationMember,
              data as ControlPlaneUpdateOrganizationMemberRequest,
            )
          ).data,
      }),

      managementTokenCreateMutation: (organization: string) => ({
        mutationKey: ['management-token:create', organization] as const,
        mutationFn: async (data: ManagementTokenCreateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.managementTokenCreate(organization, data)
              : cloudApi.managementTokenCreate(organization, data))
          ).data,
      }),

      managementTokenDeleteMutation: (managementToken: string) => ({
        mutationKey: ['management-token:delete', managementToken] as const,
        mutationFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.managementTokenDelete(managementToken)
              : cloudApi.managementTokenDelete(managementToken))
          ).data,
      }),

      organizationInviteAcceptMutation: () => ({
        mutationKey: ['organization-invite:accept'] as const,
        mutationFn: async (data: OrganizationInviteAcceptRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationInviteAccept(data)
              : cloudApi.organizationInviteAccept(data))
          ).data,
      }),

      organizationInviteRejectMutation: () => ({
        mutationKey: ['organization-invite:reject'] as const,
        mutationFn: async (data: OrganizationInviteRejectRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationInviteReject(data)
              : cloudApi.organizationInviteReject(data))
          ).data,
      }),

      organizationInviteCreateMutation: (organization: string) => ({
        mutationKey: ['organization-invite:create', organization] as const,
        mutationFn: async (data: OrganizationInviteCreateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationInviteCreate(organization, data)
              : cloudApi.organizationInviteCreate(organization, data))
          ).data,
      }),

      organizationInviteDeleteMutation: (organizationInvite: string) => ({
        mutationKey: [
          'organization-invite:delete',
          organizationInvite,
        ] as const,
        mutationFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationInviteDelete(organizationInvite)
              : cloudApi.organizationInviteDelete(organizationInvite))
          ).data,
      }),

      tenantTagsGetQuery: (organization: string, tenant: string) => ({
        queryKey: [
          'organization-tenant:list:tags',
          organization,
          tenant,
        ] as const,
        queryFn: async () =>
          (
            await controlPlaneApi.organizationTenantListTags(
              organization,
              tenant,
            )
          ).data,
      }),

      tenantTagsSetMutation: (organization: string, tenant: string) => ({
        mutationKey: [
          'organization-tenant:set:tags',
          organization,
          tenant,
        ] as const,
        mutationFn: async (tags: string[]) =>
          (
            await controlPlaneApi.organizationTenantSetTags(
              organization,
              tenant,
              { tags },
            )
          ).data,
      }),

      // Control-plane-only: transferring a tenant between organizations has no
      // legacy-cloud-API equivalent. The caller must be an OWNER of both the
      // source and destination organizations -- the transfer happens
      // immediately, there is no separate acceptance step.
      tenantTransferMutation: (organization: string, tenant: string) => ({
        mutationKey: [
          'organization-tenant:transfer',
          organization,
          tenant,
        ] as const,
        mutationFn: async (data: { destinationOrganizationId: string }) =>
          (
            await controlPlaneApi.organizationTenantTransfer(
              organization,
              tenant,
              data,
            )
          ).data,
      }),

      // Lists which of the tenant's current members would be newly added to
      // destinationOrganizationId if the transfer were confirmed right now, so
      // the modal can show that list before the user commits.
      tenantTransferPreviewQuery: (
        organization: string,
        tenant: string,
        destinationOrganizationId: string,
      ) => ({
        queryKey: [
          'organization-tenant:transfer-preview',
          organization,
          tenant,
          destinationOrganizationId,
        ] as const,
        queryFn: async () =>
          (
            await controlPlaneApi.organizationTenantTransferPreview(
              organization,
              tenant,
              { destinationOrganizationId },
            )
          ).data,
      }),

      // ── User Groups ─────────────────────────────────────────────────────────

      userGroupsListQuery: (organization: string) => ({
        queryKey: ['organization:user-groups:list', organization] as const,
        queryFn: async () =>
          (await controlPlaneApi.organizationUserGroupsList(organization)).data,
      }),

      userGroupCreateMutation: (organization: string) => ({
        mutationKey: ['organization:user-groups:create', organization] as const,
        mutationFn: async (data: {
          name: string;
          role: string;
          canViewPayloads?: boolean;
        }) =>
          (
            await controlPlaneApi.organizationUserGroupsCreate(organization, {
              name: data.name,
              role: data.role as import('@/lib/api/generated/control-plane/data-contracts').TenantMemberRoleType,
              canViewPayloads: data.canViewPayloads,
            })
          ).data,
      }),

      userGroupGetQuery: (organization: string, userGroup: string) => ({
        queryKey: [
          'organization:user-group:get',
          organization,
          userGroup,
        ] as const,
        queryFn: async () =>
          (
            await controlPlaneApi.organizationUserGroupGet(
              organization,
              userGroup,
            )
          ).data,
      }),

      userGroupUpdateMutation: (organization: string, userGroup: string) => ({
        mutationKey: [
          'organization:user-group:update',
          organization,
          userGroup,
        ] as const,
        mutationFn: async (data: {
          name?: string;
          role?: string;
          canViewPayloads?: boolean;
        }) =>
          (
            await controlPlaneApi.organizationUserGroupUpdate(
              organization,
              userGroup,
              data as import('@/lib/api/generated/control-plane/data-contracts').UpdateUserGroupRequest,
            )
          ).data,
      }),

      userGroupDeleteMutation: (organization: string, userGroup: string) => ({
        mutationKey: [
          'organization:user-group:delete',
          organization,
          userGroup,
        ] as const,
        mutationFn: async () =>
          controlPlaneApi.organizationUserGroupDelete(organization, userGroup),
      }),

      userGroupTagsGetQuery: (organization: string, userGroup: string) => ({
        queryKey: [
          'organization:user-group:tags:get',
          organization,
          userGroup,
        ] as const,
        queryFn: async () =>
          (
            await controlPlaneApi.organizationUserGroupListTags(
              organization,
              userGroup,
            )
          ).data,
      }),

      userGroupTagsSetMutation: (organization: string, userGroup: string) => ({
        mutationKey: [
          'organization:user-group:tags:set',
          organization,
          userGroup,
        ] as const,
        mutationFn: async (tags: string[]) =>
          (
            await controlPlaneApi.organizationUserGroupSetTags(
              organization,
              userGroup,
              { tags },
            )
          ).data,
      }),

      userGroupMembersListQuery: (organization: string, userGroup: string) => ({
        queryKey: [
          'organization:user-group:members:list',
          organization,
          userGroup,
        ] as const,
        queryFn: async () =>
          (
            await controlPlaneApi.organizationUserGroupListMembers(
              organization,
              userGroup,
            )
          ).data,
      }),

      userGroupMemberAddMutation: (
        organization: string,
        userGroup: string,
      ) => ({
        mutationKey: [
          'organization:user-group:members:add',
          organization,
          userGroup,
        ] as const,
        mutationFn: async (organizationMemberId: string) =>
          controlPlaneApi.organizationUserGroupAddMember(
            organization,
            userGroup,
            { organizationMemberId },
          ),
      }),

      userGroupMemberRemoveMutation: (
        organization: string,
        userGroup: string,
      ) => ({
        mutationKey: [
          'organization:user-group:members:remove',
          organization,
          userGroup,
        ] as const,
        mutationFn: async (organizationMemberId: string) =>
          controlPlaneApi.organizationUserGroupRemoveMember(
            organization,
            userGroup,
            organizationMemberId,
          ),
      }),

      organizationTenantMembersAddMutation: (
        organization: string,
        tenant: string,
      ) => ({
        mutationKey: [
          'organization-tenant:members:add',
          organization,
          tenant,
        ] as const,
        mutationFn: async (data: OrganizationTenantMembersAddRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.organizationTenantMembersAdd(
                  organization,
                  tenant,
                  data,
                )
              : cloudApi.organizationTenantMembersAdd(
                  organization,
                  tenant,
                  data,
                ))
          ).data,
      }),
    }),
    [isControlPlaneEnabled],
  );
}
