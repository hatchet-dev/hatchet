import { createContext, useContext, useCallback, useMemo } from 'react';
import api, { PaginationResponse, V1Event } from '@/lib/api';
import { useQuery, useQueryClient } from '@tanstack/react-query';
import { useCurrentTenantId } from './use-tenant';
import {
  PaginationProvider,
  PaginationProviderProps,
  usePagination,
} from './utils/use-pagination';
import { FilterProvider, useFilters } from './utils/use-filters';
import { TimeFilterProvider, useTimeFilters } from './utils/use-time-filters';

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
  initialPagination?: PaginationProviderProps;
  initialTimeRange?: {
    startTime?: string;
    endTime?: string;
    activePreset?: '30m' | '1h' | '6h' | '24h' | '7d';
  };
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
  const { tenantId } = useCurrentTenantId();
  const queryClient = useQueryClient();
  const pagination = usePagination();
  const filters = useFilters<EventsFilters>();
  const timeFilters = useTimeFilters();

  const eventsQuery = useQuery({
    queryKey: [
      'v1:events:list',
      tenantId,
      pagination,
      filters,
      timeFilters.filters,
    ],
    queryFn: async () => {
      try {
        return (
          await api.v1EventList(tenantId, {
            offset: pagination.pageSize * (pagination.currentPage - 1),
            limit: pagination.pageSize,
            keys: filters.filters.keys,
            since: timeFilters.filters.startTime,
            until: timeFilters.filters.endTime,
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
      queryKey: ['v1:events:list', tenantId, pagination, filters],
    });
  }, [queryClient, tenantId, pagination, filters]);

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
    <FilterProvider initialFilters={{}}>
      <TimeFilterProvider
        initialTimeRange={{
          activePreset: '24h',
        }}
      >
        <PaginationProvider initialPage={1} initialPageSize={50}>
          <EventsProviderContent refetchInterval={refetchInterval}>
            {children}
          </EventsProviderContent>
        </PaginationProvider>
      </TimeFilterProvider>
    </FilterProvider>
  );
}
