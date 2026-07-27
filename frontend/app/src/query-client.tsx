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

      // cancelRefetch: false — a token fetch mid-retry-backoff must not be
      // cancelled and restarted from attempt one by the next trigger.
      if (shouldForceRefresh) {
        void queryClient.invalidateQueries(
          {
            queryKey: [EXCHANGE_TOKEN_QUERY_KEY_PREFIX],
            refetchType: 'all',
          },
          { cancelRefetch: false },
        );
        return;
      }

      void queryClient.refetchQueries(
        {
          queryKey: [EXCHANGE_TOKEN_QUERY_KEY_PREFIX],
          stale: true,
          type: 'all',
        },
        { cancelRefetch: false },
      );
    }, 250);
  };

  const refreshStaleExchangeTokens = () => refreshExchangeTokens(false);
  const forceRefreshExchangeTokens = () => refreshExchangeTokens(true);

  // Tokens go stale 60s before expiry; polling well inside that window means
  // renewal happens in the background instead of blocking an interactive
  // request on a cross-region control-plane round trip. Hidden tabs skip the
  // poll — the visibilitychange handler below refreshes on return instead.
  window.setInterval(() => {
    if (document.visibilityState === 'visible') {
      refreshStaleExchangeTokens();
    }
  }, 30_000);

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
