import api, {
  ListSNSIntegrations,
  SNSIntegration,
  CreateSNSIntegrationRequest,
} from '@/lib/api';
import {
  useMutation,
  UseMutationResult,
  useQuery,
  UseQueryResult,
} from '@tanstack/react-query';
import useTenant from './use-tenant';
import {
  createContext,
  useContext,
  PropsWithChildren,
  createElement,
  useMemo,
} from 'react';

// Main hook return type
interface IngestorsState {
  sns: {
    list: UseQueryResult<ListSNSIntegrations, Error>;
    create: UseMutationResult<
      SNSIntegration,
      Error,
      CreateSNSIntegrationRequest,
      unknown
    >;
    remove: UseMutationResult<void, Error, SNSIntegration, unknown>;
  };
}

interface IngestorsProviderProps extends PropsWithChildren {
  refetchInterval?: number;
}

const IngestorsContext = createContext<IngestorsState | null>(null);

export function useIngestors() {
  const context = useContext(IngestorsContext);
  if (!context) {
    throw new Error('useIngestors must be used within a IngestorsProvider');
  }
  return context;
}

function IngestorsProviderContent({ children }: IngestorsProviderProps) {
  const { tenant } = useTenant();

  const listSNSIntegrationsQuery = useQuery({
    queryKey: ['sns:list', tenant],
    queryFn: async () => (await api.snsList(tenant?.metadata.id || '')).data,
  });

  const createSNSIntegrationMutation = useMutation({
    mutationKey: ['sns:create', tenant],
    mutationFn: async (data: CreateSNSIntegrationRequest) => {
      const res = await api.snsCreate(tenant?.metadata.id || '', data);
      return res.data;
    },
    onSuccess: () => {
      listSNSIntegrationsQuery.refetch();
    },
  });

  const deleteSNSIntegrationMutation = useMutation({
    mutationKey: ['sns:delete', tenant],
    mutationFn: async (snsIntegration: SNSIntegration) => {
      await api.snsDelete(snsIntegration.metadata.id);
    },
    onSuccess: () => {
      listSNSIntegrationsQuery.refetch();
    },
  });

  const value = useMemo(
    () => ({
      sns: {
        list: listSNSIntegrationsQuery,
        create: createSNSIntegrationMutation,
        remove: deleteSNSIntegrationMutation,
      },
    }),
    [
      listSNSIntegrationsQuery,
      createSNSIntegrationMutation,
      deleteSNSIntegrationMutation,
    ],
  );

  return createElement(IngestorsContext.Provider, { value }, children);
}

export function IngestorsProvider({
  children,
  refetchInterval,
}: IngestorsProviderProps) {
  return (
    <IngestorsProviderContent refetchInterval={refetchInterval}>
      {children}
    </IngestorsProviderContent>
  );
}
