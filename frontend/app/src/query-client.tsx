import { defaultQueryRetry } from '@/lib/query-retry';
import { QueryClient } from '@tanstack/react-query';

const queryClient: QueryClient = new QueryClient({
  defaultOptions: {
    queries: {
      retry: defaultQueryRetry,
    },
  },
});

export default queryClient;
