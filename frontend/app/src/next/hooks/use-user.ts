import api, {
  TenantMember,
  User,
  UserLoginRequest,
  UserRegisterRequest,
  TenantInvite,
} from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
} from '@tanstack/react-query';
import { AxiosResponse } from 'axios';
import { useToast } from './utils/use-toast';

interface UserState {
  data?: User;
  memberships?: TenantMember[];
  invites: {
    list: TenantInvite[];
    loading: boolean;
    accept: UseMutationResult<AxiosResponse<void, any>, Error, string, unknown>;
    reject: UseMutationResult<AxiosResponse<void, any>, Error, string, unknown>;
  };
  isLoading: boolean;
  logout: UseMutationResult<AxiosResponse<User, any>, Error, void, unknown>;
  login: UseMutationResult<User, Error, UserLoginRequest, unknown>;
  register: UseMutationResult<User, Error, UserRegisterRequest, unknown>;
}

interface UseUserOptions {
  refetchInterval?: number;
}

export default function useUser({
  refetchInterval,
}: UseUserOptions = {}): UserState {
  const { toast } = useToast();

  const userQuery = useQuery({
    queryKey: ['user:get'],
    queryFn: async () => {
      try {
        const response = await api.userGetCurrent();
        if (response.status === 403) {
          throw new Error('Forbidden');
        }
        return response.data;
      } catch (error) {
        toast({
          title: 'Error fetching user data',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    retry: 0,
    refetchInterval,
  });

  const membershipsQuery = useQuery({
    queryKey: ['user:memberships:list'],
    queryFn: async () => {
      try {
        return (await api.tenantMembershipsList()).data;
      } catch (error) {
        toast({
          title: 'Error fetching memberships',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    enabled: !!userQuery.data && userQuery.data.emailVerified,
  });

  // Query to fetch user invites with 60 second refetch interval
  const invitesQuery = useQuery({
    queryKey: ['user:list:tenant-invites'],
    queryFn: async () => {
      try {
        return (await api.userListTenantInvites()).data;
      } catch (error) {
        toast({
          title: 'Error fetching invites',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    enabled: !!userQuery.data && userQuery.data.emailVerified,
    refetchInterval: refetchInterval || 20 * 1000, // 60 seconds
  });

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: async () => {
      try {
        return await api.userUpdateLogout();
      } catch (error) {
        toast({
          title: 'Error logging out',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      // force a page reload to ensure the user is logged out
      window.location.href = '/auth/login';
    },
  });

  const loginMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: UserLoginRequest) => {
      try {
        const user = await api.userUpdateLogin(data);
        if (user.status === 200) {
          await userQuery.refetch();
        }
        return user.data;
      } catch (error) {
        toast({
          title: 'Error logging in',
          description: 'Please check your credentials and try again',
          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: (req) => {
      return req;
    },
  });

  const registerMutation = useMutation({
    mutationKey: ['user:create'],
    mutationFn: async (data: UserRegisterRequest) => {
      try {
        const user = await api.userCreate(data);
        if (user.status === 200) {
          await userQuery.refetch();
        }
        return user.data;
      } catch (error) {
        toast({
          title: 'Error registering user',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: (req) => {
      return req;
    },
  });

  const acceptInviteMutation = useMutation({
    mutationKey: ['tenant-invite:accept'],
    mutationFn: async (inviteId: string) => {
      try {
        return await api.tenantInviteAccept({ invite: inviteId });
      } catch (error) {
        toast({
          title: 'Error accepting invite',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: async () => {
      await Promise.all([
        invitesQuery.refetch(),
        userQuery.refetch(),
        membershipsQuery.refetch(),
      ]);
      return true;
    },
  });

  const rejectInviteMutation = useMutation({
    mutationKey: ['tenant-invite:reject'],
    mutationFn: async (inviteId: string) => {
      try {
        return await api.tenantInviteReject({ invite: inviteId });
      } catch (error) {
        toast({
          title: 'Error rejecting invite',

          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      invitesQuery.refetch();
    },
  });

  return {
    data: userQuery.data,
    memberships: membershipsQuery.data?.rows,
    invites: {
      list: invitesQuery.data?.rows || [],
      loading: invitesQuery.isLoading,
      accept: acceptInviteMutation,
      reject: rejectInviteMutation,
    },
    isLoading: userQuery.isLoading || membershipsQuery.isLoading,
    logout: logoutMutation,
    login: loginMutation,
    register: registerMutation,
  };
}
