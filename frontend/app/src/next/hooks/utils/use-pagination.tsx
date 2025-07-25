import * as React from 'react';
import { useStateAdapter } from '../../lib/utils/storage-adapter';

export interface PaginationManager {
  currentPage: number;
  pageSize: number;
  numPages: number;
  setCurrentPage: (page: number) => void;
  setPageSize: (size: number) => void;
  setNumPages: (numPages: number) => void;
  pageSizeOptions?: number[];
}

export const PaginationManagerNoOp: PaginationManager = {
  currentPage: 1,
  pageSize: 10,
  numPages: 1,
  setCurrentPage: () => {},
  setPageSize: () => {},
  setNumPages: () => {},
  pageSizeOptions: [10, 50, 100, 500],
};

interface PaginationContextType extends PaginationManager {}

const PaginationContext = React.createContext<
  PaginationContextType | undefined
>(undefined);

/**
 * Hook to access the current pagination state and pagination management functions
 * @returns PaginationManager instance with pagination state and functions
 */
export function usePagination() {
  const context = React.useContext(PaginationContext);
  if (!context) {
    throw new Error('usePagination must be used within a PaginationProvider');
  }
  return context;
}

// Storage type for pagination - either in-memory state or URL query parameters
type PaginationType = 'state' | 'query';

export interface PaginationProviderProps {
  initialPage?: number;
  initialPageSize?: number;
  pageSizeOptions?: number[];
  /**
   * Storage type for pagination:
   * - 'query': Store pagination in URL query parameters (default)
   * - 'state': Store pagination in component state
   */
  type?: PaginationType;
}

// Define the type for pagination state
interface PaginationState extends Record<string, number> {
  page: number;
  pageSize: number;
}

const defaultPageSizeOptions = [10, 50, 100, 500];

/**
 * Provider component that manages pagination state
 * @param props.children - React children
 * @param props.initialPage - Initial page number (default: 1)
 * @param props.initialPageSize - Initial page size (default: 50)
 * @param props.pageSizeOptions - Available page size options
 * @param props.type - Storage type ('query' or 'state')
 */
export function PaginationProvider({
  children,
  initialPage = 1,
  initialPageSize = 50,
  pageSizeOptions = defaultPageSizeOptions,
}: PaginationProviderProps & React.PropsWithChildren) {
  const state = useStateAdapter<PaginationState>({
    page: initialPage,
    pageSize: initialPageSize,
  });

  const [numPages, setNumPages] = React.useState(1);

  const currentPage = state.getValue('page', initialPage);
  const pageSize = state.getValue('pageSize', initialPageSize);

  const setCurrentPage = React.useCallback(
    (page: number) => {
      state.setValue('page', page);
    },
    [state],
  );

  const handlePageSizeChange = React.useCallback(
    (newPageSize: number) => {
      state.setValue('pageSize', newPageSize);

      const newNumPages = Math.ceil((currentPage * pageSize) / newPageSize);

      if (currentPage > newNumPages) {
        state.setValue('page', Math.min(currentPage, newNumPages));
      }
    },
    [currentPage, pageSize, state],
  );

  const value = React.useMemo(
    () => ({
      currentPage,
      pageSize,
      numPages,
      setCurrentPage,
      setPageSize: handlePageSizeChange,
      setNumPages,
      pageSizeOptions,
    }),
    [
      currentPage,
      pageSize,
      numPages,
      pageSizeOptions,
      handlePageSizeChange,
      setCurrentPage,
    ],
  );

  return (
    <PaginationContext.Provider value={value}>
      {children}
    </PaginationContext.Provider>
  );
}
