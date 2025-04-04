import { queries, User } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

interface UserState {
  data?: User;
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

  return {
    data: userQuery.data,
    isLoading: false,
  };
}
