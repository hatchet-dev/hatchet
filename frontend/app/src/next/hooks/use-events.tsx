import { createContext, useContext, useCallback, useMemo } from 'react';
import api, { PaginationResponse, V1Event } from '@/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import useTenant from './use-tenant';
import { PaginationProvider, usePagination } from './utils/use-pagination';
import { FilterProvider, useFilters } from './utils/use-filters';

interface EventsState {
  data: V1Event[];
  paginationData?: { current_page: number; num_pages: number };
  isLoading: boolean;
  invalidate: () => Promise<void>;
  pagination: ReturnType<typeof usePagination>;
}

interface EventsProviderProps {
  children: React.ReactNode;
  refetchInterval?: number;
}

export interface EventsFilters {
  keys?: string[];
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
  const filters = useFilters<EventsFilters>();

  const eventsQuery = useQuery({
    queryKey: ['v1:events:list', tenant, pagination, filters],
    queryFn: async () => {
      try {
        return (
          await api.v1EventList(tenant?.metadata.id || '', {
            offset: pagination.pageSize * (pagination.currentPage - 1),
            limit: pagination.pageSize,
            keys: filters.filters.keys,
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
      queryKey: ['v1:events:list', tenant, pagination, filters],
    });
  }, [queryClient, tenant?.metadata.id, pagination, filters]);

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

export function EventsProvider({
  children,
  refetchInterval,
}: EventsProviderProps) {
  return (
    <FilterProvider initialFilters={{}} type="state">
      <PaginationProvider initialPage={1} initialPageSize={50}>
        <EventsProviderContent refetchInterval={refetchInterval}>
          {children}
        </EventsProviderContent>
      </PaginationProvider>
    </FilterProvider>
  );
}
