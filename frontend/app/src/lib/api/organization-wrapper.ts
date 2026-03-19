import useControlPlane from '@/hooks/use-control-plane';
import { cloudApi, controlPlaneApi } from '@/lib/api/api';
import { useMemo } from 'react';

type OrganizationCreateRequest = Parameters<
  typeof cloudApi.organizationCreate
>[0];
type OrganizationUpdateRequest = Parameters<
  typeof cloudApi.organizationUpdate
>[1];
type OrganizationCreateTenantRequest = Parameters<
  typeof cloudApi.organizationCreateTenant
>[1];
type OrganizationMemberDeleteRequest = Parameters<
  typeof cloudApi.organizationMemberDelete
>[1];
type ManagementTokenCreateRequest = Parameters<
  typeof cloudApi.managementTokenCreate
>[1];
type OrganizationInviteAcceptRequest = Parameters<
  typeof cloudApi.organizationInviteAccept
>[0];
type OrganizationInviteRejectRequest = Parameters<
  typeof cloudApi.organizationInviteReject
>[0];
type OrganizationInviteCreateRequest = Parameters<
  typeof cloudApi.organizationInviteCreate
>[1];

export function useOrganizationApi() {
  const { isControlPlaneEnabled } = useControlPlane();

  return useMemo(
    () => ({
      organizationList: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationList()
          : cloudApi.organizationList(),

      organizationCreate: (data: OrganizationCreateRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationCreate(data)
          : cloudApi.organizationCreate(data),

      organizationGet: (organization: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationGet(organization)
          : cloudApi.organizationGet(organization),

      organizationUpdate: (
        organization: string,
        data: OrganizationUpdateRequest,
      ) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationUpdate(organization, data)
          : cloudApi.organizationUpdate(organization, data),

      organizationCreateTenant: (
        organization: string,
        data: OrganizationCreateTenantRequest,
      ) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationCreateTenant(organization, data)
          : cloudApi.organizationCreateTenant(organization, data),

      organizationTenantDelete: (organizationTenant: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationTenantDelete(organizationTenant)
          : cloudApi.organizationTenantDelete(organizationTenant),

      organizationMemberDelete: (
        organizationMember: string,
        data: OrganizationMemberDeleteRequest,
      ) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationMemberDelete(organizationMember, data)
          : cloudApi.organizationMemberDelete(organizationMember, data),

      managementTokenCreate: (
        organization: string,
        data: ManagementTokenCreateRequest,
      ) =>
        isControlPlaneEnabled
          ? controlPlaneApi.managementTokenCreate(organization, data)
          : cloudApi.managementTokenCreate(organization, data),

      managementTokenList: (organization: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.managementTokenList(organization)
          : cloudApi.managementTokenList(organization),

      managementTokenDelete: (managementToken: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.managementTokenDelete(managementToken)
          : cloudApi.managementTokenDelete(managementToken),

      userListOrganizationInvites: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.userListOrganizationInvites()
          : cloudApi.userListOrganizationInvites(),

      organizationInviteAccept: (data: OrganizationInviteAcceptRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationInviteAccept(data)
          : cloudApi.organizationInviteAccept(data),

      organizationInviteReject: (data: OrganizationInviteRejectRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationInviteReject(data)
          : cloudApi.organizationInviteReject(data),

      organizationInviteList: (organization: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationInviteList(organization)
          : cloudApi.organizationInviteList(organization),

      organizationInviteCreate: (
        organization: string,
        data: OrganizationInviteCreateRequest,
      ) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationInviteCreate(organization, data)
          : cloudApi.organizationInviteCreate(organization, data),

      organizationInviteDelete: (organizationInvite: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.organizationInviteDelete(organizationInvite)
          : cloudApi.organizationInviteDelete(organizationInvite),
    }),
    [isControlPlaneEnabled],
  );
}
