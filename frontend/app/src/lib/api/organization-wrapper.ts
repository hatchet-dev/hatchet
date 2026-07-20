import { controlPlaneApi } from '@/lib/api/api';
import type {
  CreateNewTenantForOrganizationRequest,
  CreateOrganizationInviteRequest,
  OrganizationMemberRoleType,
  UpdateOrganizationMemberRequest,
} from '@/lib/api/generated/control-plane/data-contracts';

type OrganizationCreateRequest = Parameters<
  typeof controlPlaneApi.organizationCreate
>[0];
type OrganizationUpdateRequest = {
  name?: string;
  inactivity_timeout?: string;
};
type OrganizationCreateTenantRequest = CreateNewTenantForOrganizationRequest;
type OrganizationMemberDeleteRequest = Parameters<
  typeof controlPlaneApi.organizationMemberDelete
>[1];
type OrganizationMemberUpdateRequest = {
  role: `${OrganizationMemberRoleType}`;
};
type ManagementTokenCreateRequest = Parameters<
  typeof controlPlaneApi.managementTokenCreate
>[1];
type OrganizationInviteAcceptRequest = Parameters<
  typeof controlPlaneApi.organizationInviteAccept
>[0];
type OrganizationInviteRejectRequest = Parameters<
  typeof controlPlaneApi.organizationInviteReject
>[0];
export type OrganizationInviteCreateRequest = CreateOrganizationInviteRequest;
type OrganizationTenantMembersAddRequest = Parameters<
  typeof controlPlaneApi.organizationTenantMembersAdd
>[2];

export function useOrganizationApi() {
  return {
    organizationListQuery: () => ({
      queryKey: ['organization:list'] as const,
      queryFn: async () => (await controlPlaneApi.organizationList()).data,
    }),

    organizationGetQuery: (organization: string) => ({
      queryKey: ['organization:get', organization] as const,
      queryFn: async () =>
        (await controlPlaneApi.organizationGet(organization)).data,
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
        (await controlPlaneApi.organizationEntitlementsGet(organization)).data,
    }),

    managementTokenListQuery: (organization: string) => ({
      queryKey: ['management-tokens:list', organization] as const,
      queryFn: async () =>
        (await controlPlaneApi.managementTokenList(organization)).data,
    }),

    userListOrganizationInvitesQuery: () => ({
      queryKey: ['user:list:organization-invites'] as const,
      queryFn: async () =>
        (await controlPlaneApi.userListOrganizationInvites()).data,
    }),

    organizationInviteListQuery: (organization: string) => ({
      queryKey: ['organization-invites:list', organization] as const,
      queryFn: async () =>
        (await controlPlaneApi.organizationInviteList(organization)).data,
    }),

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
        (await controlPlaneApi.organizationCreate(data)).data,
    }),

    organizationUpdateMutation: (organization: string) => ({
      mutationKey: ['organization:update', organization] as const,
      mutationFn: async (data: OrganizationUpdateRequest) =>
        (await controlPlaneApi.organizationUpdate(organization, data)).data,
    }),

    organizationCreateTenantMutation: (organization: string) => ({
      mutationKey: ['organization:create-tenant', organization] as const,
      mutationFn: async (data: OrganizationCreateTenantRequest) =>
        (await controlPlaneApi.organizationCreateTenant(organization, data))
          .data,
    }),

    organizationTenantDeleteMutation: (organizationTenant: string) => ({
      mutationKey: ['organization-tenant:delete', organizationTenant] as const,
      mutationFn: async () =>
        (await controlPlaneApi.organizationTenantDelete(organizationTenant))
          .data,
    }),

    organizationMemberDeleteMutation: (organizationMember: string) => ({
      mutationKey: ['organization-member:delete', organizationMember] as const,
      mutationFn: async (data: OrganizationMemberDeleteRequest) =>
        (
          await controlPlaneApi.organizationMemberDelete(
            organizationMember,
            data,
          )
        ).data,
    }),

    organizationMemberUpdateMutation: (organizationMember: string) => ({
      mutationKey: ['organization-member:update', organizationMember] as const,
      mutationFn: async (data: OrganizationMemberUpdateRequest) =>
        (
          await controlPlaneApi.organizationMemberUpdate(
            organizationMember,
            data as UpdateOrganizationMemberRequest,
          )
        ).data,
    }),

    managementTokenCreateMutation: (organization: string) => ({
      mutationKey: ['management-token:create', organization] as const,
      mutationFn: async (data: ManagementTokenCreateRequest) =>
        (await controlPlaneApi.managementTokenCreate(organization, data)).data,
    }),

    managementTokenDeleteMutation: (managementToken: string) => ({
      mutationKey: ['management-token:delete', managementToken] as const,
      mutationFn: async () =>
        (await controlPlaneApi.managementTokenDelete(managementToken)).data,
    }),

    organizationInviteAcceptMutation: () => ({
      mutationKey: ['organization-invite:accept'] as const,
      mutationFn: async (data: OrganizationInviteAcceptRequest) =>
        (await controlPlaneApi.organizationInviteAccept(data)).data,
    }),

    organizationInviteRejectMutation: () => ({
      mutationKey: ['organization-invite:reject'] as const,
      mutationFn: async (data: OrganizationInviteRejectRequest) =>
        (await controlPlaneApi.organizationInviteReject(data)).data,
    }),

    organizationInviteCreateMutation: (organization: string) => ({
      mutationKey: ['organization-invite:create', organization] as const,
      mutationFn: async (data: OrganizationInviteCreateRequest) =>
        (await controlPlaneApi.organizationInviteCreate(organization, data))
          .data,
    }),

    organizationInviteDeleteMutation: (organizationInvite: string) => ({
      mutationKey: ['organization-invite:delete', organizationInvite] as const,
      mutationFn: async () =>
        (await controlPlaneApi.organizationInviteDelete(organizationInvite))
          .data,
    }),

    tenantTagsGetQuery: (organization: string, tenant: string) => ({
      queryKey: [
        'organization-tenant:list:tags',
        organization,
        tenant,
      ] as const,
      queryFn: async () =>
        (await controlPlaneApi.organizationTenantListTags(organization, tenant))
          .data,
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

    userGroupsListQuery: (organization: string) => ({
      queryKey: ['organization:user-groups:list', organization] as const,
      queryFn: async () =>
        (await controlPlaneApi.organizationUserGroupsList(organization)).data,
    }),

    userGroupCreateMutation: (organization: string) => ({
      mutationKey: ['organization:user-groups:create', organization] as const,
      mutationFn: async (data: { name: string; role: string }) =>
        (
          await controlPlaneApi.organizationUserGroupsCreate(organization, {
            name: data.name,
            role: data.role as import('@/lib/api/generated/control-plane/data-contracts').TenantMemberRoleType,
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
      mutationFn: async (data: { name?: string; role?: string }) =>
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

    userGroupMemberAddMutation: (organization: string, userGroup: string) => ({
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
          await controlPlaneApi.organizationTenantMembersAdd(
            organization,
            tenant,
            data,
          )
        ).data,
    }),
  };
}
