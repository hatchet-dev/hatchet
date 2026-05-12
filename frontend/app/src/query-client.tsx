import { EXCHANGE_TOKEN_QUERY_KEY_PREFIX } from '@/lib/api/exchange-token';
import { defaultQueryRetry } from '@/lib/query-retry';
import { QueryClient } from '@tanstack/react-query';

const queryClient: QueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: defaultQueryRetry,
    },
  },
});

type NetworkInformationLike = EventTarget;

type NavigatorWithConnection = Navigator & {
  connection?: NetworkInformationLike;
  mozConnection?: NetworkInformationLike;
  webkitConnection?: NetworkInformationLike;
};

function setupExchangeTokenRefreshTriggers() {
  if (typeof window === 'undefined' || typeof navigator === 'undefined') {
    return;
  }

  let refreshTimeout: number | undefined;
  let forceRefresh = false;

  const refreshExchangeTokens = (force: boolean) => {
    forceRefresh = forceRefresh || force;

    if (refreshTimeout) {
      window.clearTimeout(refreshTimeout);
    }

    refreshTimeout = window.setTimeout(() => {
      refreshTimeout = undefined;
      const shouldForceRefresh = forceRefresh;
      forceRefresh = false;

      if (shouldForceRefresh) {
        void queryClient.invalidateQueries({
          queryKey: [EXCHANGE_TOKEN_QUERY_KEY_PREFIX],
          refetchType: 'all',
        });
        return;
      }

      void queryClient.refetchQueries({
        queryKey: [EXCHANGE_TOKEN_QUERY_KEY_PREFIX],
        stale: true,
        type: 'all',
      });
    }, 250);
  };

  const refreshStaleExchangeTokens = () => refreshExchangeTokens(false);
  const forceRefreshExchangeTokens = () => refreshExchangeTokens(true);

  window.addEventListener('focus', refreshStaleExchangeTokens);
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'visible') {
      refreshStaleExchangeTokens();
    }
  });
  window.addEventListener('online', forceRefreshExchangeTokens);

  const navigatorWithConnection = navigator as NavigatorWithConnection;
  const connection =
    navigatorWithConnection.connection ??
    navigatorWithConnection.mozConnection ??
    navigatorWithConnection.webkitConnection;

  connection?.addEventListener('change', forceRefreshExchangeTokens);
}

setupExchangeTokenRefreshTriggers();

export default queryClient;
