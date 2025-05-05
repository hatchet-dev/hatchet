import { createContext, useContext, useCallback, useMemo } from 'react';
import api, { Event, PaginationResponse } from '@/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import useTenant from './use-tenant';
import { PaginationProvider, usePagination } from './utils/use-pagination';

interface EventsState {
  data: Event[];
  paginationData?: { current_page: number; num_pages: number };
  isLoading: boolean;
  invalidate: () => Promise<void>;
  pagination: ReturnType<typeof usePagination>;
}

interface EventsProviderProps {
  children: React.ReactNode;
  refetchInterval?: number;
}

const EventsContext = createContext<EventsState | null>(null);

export function useEvents() {
  const context = useContext(EventsContext);
  if (!context) {
    throw new Error('useEvents must be used within a EventsProvider');
  }
  return context;
}

function EventsProviderContent({ children }: EventsProviderProps) {
  const { tenant } = useTenant();
  const queryClient = useQueryClient();
  const pagination = usePagination();

  const eventsQuery = useQuery({
    queryKey: ['v1:events:list', tenant, pagination],
    queryFn: async () => {
      try {
        return (
          await api.v1EventList(tenant?.metadata.id || '', {
            offset: 0,
            limit: 10,
          })
        ).data;
      } catch (error) {
        return {
          rows: [],
          pagination: {
            current_page: 1,
            num_pages: 1,
          } as PaginationResponse,
        };
      }
    },
  });

  const invalidate = useCallback(async () => {
    await queryClient.invalidateQueries({
      queryKey: ['v1:events:list', tenant, pagination],
    });
  }, [queryClient, tenant?.metadata.id, pagination]);

  const value = useMemo(
    () => ({
      data: eventsQuery.data?.rows || [],
      paginationData:
        eventsQuery.data?.pagination ||
        ({
          current_page: 1,
          num_pages: 1,
        } as PaginationResponse),
      isLoading: eventsQuery.isLoading,
      invalidate,
      pagination,
    }),
    [eventsQuery.data, eventsQuery.isLoading, invalidate, pagination],
  );

  return (
    <EventsContext.Provider
      value={{
        ...value,
        paginationData: {
          ...value.paginationData,
          current_page: value.paginationData.current_page || 1,
          num_pages: value.paginationData.num_pages || 1,
        },
      }}
    >
      {children}
    </EventsContext.Provider>
  );
}

export function WorkflowsProvider({
  children,
  refetchInterval,
}: EventsProviderProps) {
  return (
    <PaginationProvider initialPage={1} initialPageSize={50}>
      <EventsProviderContent refetchInterval={refetchInterval}>
        {children}
      </EventsProviderContent>
    </PaginationProvider>
  );
}
