import { useAppContext } from '@/providers/app-context';

/**
 * Hook to access the current user
 *
 * Now backed by AppContext for better performance and consistency.
 * Maintains backward compatibility with the old API.
 */
export function useCurrentUser() {
  const { user, isUserLoading, isUserError, userError } = useAppContext();

  return {
    currentUser: user,
    isLoading: isUserLoading,
    isError: isUserError || (!user && !isUserLoading),
    error: userError,
  };
}
