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
      // ── Queries ────────────────────────────────────────────────────────────

      tenantMembershipsListQuery: () => ({
        queryKey: ['tenant-memberships:list'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantMembershipsList()
              : api.tenantMembershipsList())
          ).data,
      }),

      userListTenantInvitesQuery: () => ({
        queryKey: ['user:list:tenant-invites'] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.userListTenantInvites()
              : api.userListTenantInvites())
          ).data,
      }),

      tenantMemberListQuery: (tenant: string) => ({
        queryKey: ['tenant-member:list', tenant] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantMemberList(tenant)
              : api.tenantMemberList(tenant))
          ).data,
      }),

      tenantInviteListQuery: (tenant: string) => ({
        queryKey: ['tenant-invite:list', tenant] as const,
        queryFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantInviteList(tenant)
              : api.tenantInviteList(tenant))
          ).data,
      }),

      // ── Mutations ──────────────────────────────────────────────────────────

      tenantInviteRejectMutation: () => ({
        mutationKey: ['tenant-invite:reject'] as const,
        mutationFn: async (data: TenantInviteRejectRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantInviteReject(data)
              : api.tenantInviteReject(data))
          ).data,
      }),

      tenantInviteAcceptMutation: () => ({
        mutationKey: ['tenant-invite:accept'] as const,
        mutationFn: async (data: TenantInviteAcceptRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantInviteAccept(data)
              : api.tenantInviteAccept(data))
          ).data,
      }),

      tenantMemberUpdateMutation: (tenant: string, tenantMember: string) => ({
        mutationKey: ['tenant-member:update', tenant, tenantMember] as const,
        mutationFn: async (data: TenantMemberUpdateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantMemberUpdate(tenant, tenantMember, data)
              : api.tenantMemberUpdate(tenant, tenantMember, data))
          ).data,
      }),

      tenantMemberDeleteMutation: (tenant: string, tenantMember: string) => ({
        mutationKey: ['tenant-member:delete', tenant, tenantMember] as const,
        mutationFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantMemberDelete(tenant, tenantMember)
              : api.tenantMemberDelete(tenant, tenantMember))
          ).data,
      }),

      tenantInviteCreateMutation: (tenant: string) => ({
        mutationKey: ['tenant-invite:create', tenant] as const,
        mutationFn: async (data: TenantInviteCreateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantInviteCreate(tenant, data)
              : api.tenantInviteCreate(tenant, data))
          ).data,
      }),

      tenantInviteUpdateMutation: (tenant: string, tenantInvite: string) => ({
        mutationKey: ['tenant-invite:update', tenant, tenantInvite] as const,
        mutationFn: async (data: TenantInviteUpdateRequest) =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantInviteUpdate(tenant, tenantInvite, data)
              : api.tenantInviteUpdate(tenant, tenantInvite, data))
          ).data,
      }),

      tenantInviteDeleteMutation: (tenant: string, tenantInvite: string) => ({
        mutationKey: ['tenant-invite:delete', tenant, tenantInvite] as const,
        mutationFn: async () =>
          (
            await (isControlPlaneEnabled
              ? controlPlaneApi.tenantInviteDelete(tenant, tenantInvite)
              : api.tenantInviteDelete(tenant, tenantInvite))
          ).data,
      }),
    }),
    [isControlPlaneEnabled],
  );
}
