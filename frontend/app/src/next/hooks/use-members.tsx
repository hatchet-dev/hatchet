import { createContext, useContext } from 'react';
import {
  useQuery,
  useMutation,
  UseMutationResult,
} from '@tanstack/react-query';
import api from '@/lib/api';
import useTenant from './use-tenant';
import {
  TenantMember,
  TenantMemberList,
  CreateTenantInviteRequest,
  TenantInvite,
  TenantInviteList,
} from '@/lib/api/generated/data-contracts';

interface MembersState {
  data: TenantMember[];
  isLoading: boolean;
  refetch: () => Promise<unknown>;
  invite: UseMutationResult<
    TenantInvite,
    Error,
    CreateTenantInviteRequest,
    unknown
  >;
  invites: TenantInvite[];
  isLoadingInvites: boolean;
  refetchInvites: () => Promise<unknown>;
}

const MembersContext = createContext<MembersState | null>(null);

export function MembersProvider({ children }: { children: React.ReactNode }) {
  const { tenant } = useTenant();

  const membersQuery = useQuery({
    queryKey: ['tenant-members:list', tenant?.metadata.id],
    queryFn: async (): Promise<TenantMemberList> => {
      if (!tenant?.metadata.id) {
        return { rows: [] };
      }
      return (await api.tenantMemberList(tenant.metadata.id)).data;
    },
    enabled: !!tenant?.metadata.id,
  });

  const invitesQuery = useQuery({
    queryKey: ['tenant-invites:list', tenant?.metadata.id],
    queryFn: async (): Promise<TenantInviteList> => {
      if (!tenant?.metadata.id) {
        return { rows: [] };
      }
      return (await api.tenantInviteList(tenant.metadata.id)).data;
    },
    enabled: !!tenant?.metadata.id,
  });

  const inviteMutation = useMutation({
    mutationKey: ['tenant-invite:create', tenant?.metadata.id],
    mutationFn: async (data: CreateTenantInviteRequest) => {
      if (!tenant?.metadata.id) {
        throw new Error('Tenant not found');
      }
      const res = await api.tenantInviteCreate(tenant.metadata.id, data);
      return res.data;
    },
    onSuccess: () => {
      // Refresh the members list and invites list after successful invite
      membersQuery.refetch();
      invitesQuery.refetch();
    },
  });

  const value = {
    data: membersQuery.data?.rows || [],
    isLoading: membersQuery.isLoading,
    refetch: membersQuery.refetch,
    invite: inviteMutation,
    invites: invitesQuery.data?.rows || [],
    isLoadingInvites: invitesQuery.isLoading,
    refetchInvites: invitesQuery.refetch,
  };

  return (
    <MembersContext.Provider value={value}>{children}</MembersContext.Provider>
  );
}

export default function useMembers(): MembersState {
  const context = useContext(MembersContext);
  if (!context) {
    throw new Error('useMembers must be used within a MembersProvider');
  }
  return context;
}
