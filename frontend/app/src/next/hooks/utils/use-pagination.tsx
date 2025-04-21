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
interface PaginationState {
  page: number;
  pageSize: number;
}

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
  pageSizeOptions = [10, 50, 100, 500],
  type = 'query',
}: PaginationProviderProps & React.PropsWithChildren) {
  // Initialize the storage with default values
  const state = useStateAdapter<PaginationState>(
    {
      page: initialPage,
      pageSize: initialPageSize,
    },
    { type },
  );

  const [numPages, setNumPages] = React.useState(1);

  // Get current values from the storage adapter
  const currentPage = state.getValue('page', initialPage);
  const pageSize = state.getValue('pageSize', initialPageSize);

  // Set current page through the adapter
  const setCurrentPage = React.useCallback(
    (page: number) => {
      state.setValue('page', page);
    },
    [state],
  );

  // Handle page size change through the adapter
  const handlePageSizeChange = React.useCallback(
    (newPageSize: number) => {
      state.setValue('pageSize', newPageSize);

      // Calculate the new number of pages based on the new page size
      const newNumPages = Math.ceil((currentPage * pageSize) / newPageSize);

      // Ensure current page is valid
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
