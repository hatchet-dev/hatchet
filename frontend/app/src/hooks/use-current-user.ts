import { queries } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';

export function useCurrentUser() {
  const currentUserQuery = useQuery({
    ...queries.user.current,
    retry: false,
  });

  return {
    currentUser: currentUserQuery.data,
    isLoading: currentUserQuery.isLoading,
    isError: currentUserQuery.isError,
    error: currentUserQuery.error,
  };
}
