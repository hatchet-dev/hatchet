import { queries, TenantMember, User } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

interface UserState {
  data?: User;
  memberships?: TenantMember[];
  isLoading: boolean;
}

export default function useUser(): UserState {
  const userQuery = useQuery({
    ...queries.user.current,
  });

  if (userQuery.isError) {
    // TODO: handle error
    console.error(userQuery.error);
  }

  const membershipsQuery = useQuery({
    ...queries.user.listTenantMemberships,
  });

  if (membershipsQuery.isError) {
    // TODO: handle error
    console.error(membershipsQuery.error);
  }

  return {
    data: userQuery.data,
    memberships: membershipsQuery.data?.rows,
    isLoading: false,
  };
}
