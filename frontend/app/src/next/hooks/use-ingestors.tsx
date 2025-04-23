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
import { useToast } from './utils/use-toast';

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
  const { toast } = useToast();

  const listSNSIntegrationsQuery = useQuery({
    queryKey: ['sns:list', tenant],
    queryFn: async () => {
      try {
        return (await api.snsList(tenant?.metadata.id || '')).data;
      } catch (error) {
        toast({
          title: 'Error fetching SNS integrations',
          
          variant: 'destructive',
          error,
        });
        return {
          rows: [],
          pagination: { current_page: 0, num_pages: 0 },
        } as ListSNSIntegrations;
      }
    },
  });

  const createSNSIntegrationMutation = useMutation({
    mutationKey: ['sns:create', tenant],
    mutationFn: async (data: CreateSNSIntegrationRequest) => {
      try {
        const res = await api.snsCreate(tenant?.metadata.id || '', data);
        return res.data;
      } catch (error) {
        toast({
          title: 'Error creating SNS integration',
          
          variant: 'destructive',
          error,
        });
        throw error;
      }
    },
    onSuccess: () => {
      listSNSIntegrationsQuery.refetch();
    },
  });

  const deleteSNSIntegrationMutation = useMutation({
    mutationKey: ['sns:delete', tenant],
    mutationFn: async (snsIntegration: SNSIntegration) => {
      try {
        await api.snsDelete(snsIntegration.metadata.id);
      } catch (error) {
        toast({
          title: 'Error deleting SNS integration',
          
          variant: 'destructive',
          error,
        });
        throw error;
      }
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
