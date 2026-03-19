import useControlPlane from '@/hooks/use-control-plane';
import api, { controlPlaneApi } from '@/lib/api/api';
import { useMemo } from 'react';

type TenantInviteRejectRequest = Parameters<typeof api.tenantInviteReject>[0];
type TenantInviteAcceptRequest = Parameters<typeof api.tenantInviteAccept>[0];
type TenantMemberUpdateRequest = Parameters<typeof api.tenantMemberUpdate>[2];
type TenantInviteCreateRequest = Parameters<typeof api.tenantInviteCreate>[1];
type TenantInviteUpdateRequest = Parameters<typeof api.tenantInviteUpdate>[2];

export function useTenantApi() {
  const { isControlPlaneEnabled } = useControlPlane();

  return useMemo(
    () => ({
      tenantMembershipsList: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantMembershipsList()
          : api.tenantMembershipsList(),

      userListTenantInvites: () =>
        isControlPlaneEnabled
          ? controlPlaneApi.userListTenantInvites()
          : api.userListTenantInvites(),

      tenantInviteReject: (data: TenantInviteRejectRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantInviteReject(data)
          : api.tenantInviteReject(data),

      // tenantInviteAccept has no control-plane equivalent — always uses main API
      // TODO-CONTROL-PLANE: need to implement this...
      tenantInviteAccept: (data: TenantInviteAcceptRequest) =>
        api.tenantInviteAccept(data),

      tenantMemberList: (tenant: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantMemberList(tenant)
          : api.tenantMemberList(tenant),

      tenantMemberUpdate: (
        tenant: string,
        tenantMember: string,
        data: TenantMemberUpdateRequest,
      ) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantMemberUpdate(tenant, tenantMember, data)
          : api.tenantMemberUpdate(tenant, tenantMember, data),

      tenantMemberDelete: (tenant: string, tenantMember: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantMemberDelete(tenant, tenantMember)
          : api.tenantMemberDelete(tenant, tenantMember),

      tenantInviteList: (tenant: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantInviteList(tenant)
          : api.tenantInviteList(tenant),

      tenantInviteCreate: (tenant: string, data: TenantInviteCreateRequest) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantInviteCreate(tenant, data)
          : api.tenantInviteCreate(tenant, data),

      tenantInviteUpdate: (
        tenant: string,
        tenantInvite: string,
        data: TenantInviteUpdateRequest,
      ) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantInviteUpdate(tenant, tenantInvite, data)
          : api.tenantInviteUpdate(tenant, tenantInvite, data),

      tenantInviteDelete: (tenant: string, tenantInvite: string) =>
        isControlPlaneEnabled
          ? controlPlaneApi.tenantInviteDelete(tenant, tenantInvite)
          : api.tenantInviteDelete(tenant, tenantInvite),
    }),
    [isControlPlaneEnabled],
  );
}
