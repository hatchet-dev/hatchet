import api, { TenantMember, User } from '@/lib/api';
import { useMutation, useQuery } from '@tanstack/react-query';

interface UserState {
  data?: User;
  memberships?: TenantMember[];
  isLoading: boolean;
  logout: () => void;
}

interface UseUserOptions {
  refetchInterval?: number;
}

export default function useUser({
  refetchInterval,
}: UseUserOptions = {}): UserState {
  const userQuery = useQuery({
    queryKey: ['user:get'],
    queryFn: async () => {
      const response = await api.userGetCurrent();
      if (response.status === 403) {
        throw new Error('Forbidden');
      }
      return response.data;
    },
    retry: 0,
    refetchInterval,
  });

  if (userQuery.isError) {
    // TODO: handle error
    console.error(userQuery.error);
  }

  const membershipsQuery = useQuery({
    queryKey: ['tenant-memberships:list'],
    queryFn: async () => (await api.tenantMembershipsList()).data,
    enabled: !!userQuery.data && userQuery.data.emailVerified,
  });

  if (membershipsQuery.isError) {
    // TODO: handle error
    console.error(membershipsQuery.error);
  }

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      await api.userUpdateLogout();
    },
    onSuccess: () => {
      // force a page reload to ensure the user is logged out
      window.location.href = '/auth/login';
    },
  });

  return {
    data: userQuery.data,
    memberships: membershipsQuery.data?.rows,
    isLoading: userQuery.isLoading,
    logout: () => logoutMutation.mutate(),
  };
}
