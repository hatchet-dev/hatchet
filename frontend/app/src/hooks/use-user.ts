import api, {
  TenantMember,
  User,
  UserLoginRequest,
  UserRegisterRequest,
} from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
} from '@tanstack/react-query';
import { AxiosResponse } from 'axios';

interface UserState {
  data?: User;
  memberships?: TenantMember[];
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
    queryKey: ['user-memberships:list'],
    queryFn: async () => (await api.tenantMembershipsList()).data,
    enabled: !!userQuery.data && userQuery.data.emailVerified,
  });

  if (membershipsQuery.isError) {
    // TODO: handle error
    console.error(membershipsQuery.error);
  }

  const logoutMutation = useMutation({
    mutationKey: ['user:update:logout'],
    mutationFn: () => {
      return api.userUpdateLogout();
    },
    onSuccess: () => {
      // force a page reload to ensure the user is logged out
      window.location.href = '/auth/login';
    },
  });

  const loginMutation = useMutation({
    mutationKey: ['user:update:login'],
    mutationFn: async (data: UserLoginRequest) => {
      const user = await api.userUpdateLogin(data);
      if (user.status === 200) {
        await userQuery.refetch();
      }
      return user.data;
    },
    onSuccess: (req) => {
      return req;
    },
  });

  const registerMutation = useMutation({
    mutationKey: ['user:create'],
    mutationFn: async (data: UserRegisterRequest) => {
      const user = await api.userCreate(data);
      if (user.status === 200) {
        await userQuery.refetch();
      }
      return user.data;
    },
    onSuccess: (req) => {
      return req;
    },
  });

  return {
    data: userQuery.data,
    memberships: membershipsQuery.data?.rows,
    isLoading: userQuery.isLoading,
    logout: logoutMutation,
    login: loginMutation,
    register: registerMutation,
  };
}
