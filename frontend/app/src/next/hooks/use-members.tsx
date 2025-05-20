import { createContext, useContext } from 'react';
import {
  useQuery,
  useMutation,
  UseMutationResult,
} from '@tanstack/react-query';
import api from '@/lib/api';
import { useCurrentTenantId } from './use-tenant';
import {
  TenantMember,
  TenantMemberList,
  CreateTenantInviteRequest,
  TenantInvite,
  TenantInviteList,
} from '@/lib/api/generated/data-contracts';
import { useToast } from './utils/use-toast';

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
  const { tenantId } = useCurrentTenantId();
  const { toast } = useToast();

  const membersQuery = useQuery({
    queryKey: ['tenant-members:list', tenantId],
    queryFn: async (): Promise<TenantMemberList> => {
      try {
        return (await api.tenantMemberList(tenantId)).data;
      } catch (error) {
        toast({
          title: 'Error fetching members',

          variant: 'destructive',
          error,
        });
        return { rows: [] };
      }
    },
  });

  const invitesQuery = useQuery({
    queryKey: ['tenant-invites:list', tenantId],
    queryFn: async (): Promise<TenantInviteList> => {
      try {
        return (await api.tenantInviteList(tenantId)).data;
      } catch (error) {
        toast({
          title: 'Error fetching invites',

          variant: 'destructive',
          error,
        });
        return { rows: [] };
      }
    },
  });

  const inviteMutation = useMutation({
    mutationKey: ['tenant-invite:create', tenantId],
    mutationFn: async (data: CreateTenantInviteRequest) => {
      try {
        const res = await api.tenantInviteCreate(tenantId, data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating invite',

          variant: 'destructive',
          error,
        });
        throw error;
      }
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
