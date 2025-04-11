import * as React from 'react';

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

export function usePagination() {
  const context = React.useContext(PaginationContext);
  if (!context) {
    throw new Error('usePagination must be used within a PaginationProvider');
  }
  return context;
}

interface PaginationProviderProps {
  children: React.ReactNode;
  initialPage?: number;
  initialPageSize?: number;
  pageSizeOptions?: number[];
}

export function PaginationProvider({
  children,
  initialPage = 1,
  initialPageSize = 50,
  pageSizeOptions = [10, 50, 100, 500],
}: PaginationProviderProps) {
  const [currentPage, setCurrentPage] = React.useState(initialPage);
  const [pageSize, setPageSize] = React.useState(initialPageSize);
  const [numPages, setNumPages] = React.useState(1);

  const handlePageSizeChange = React.useCallback(
    (newPageSize: number) => {
      setPageSize(newPageSize);
      // Calculate the new number of pages based on the new page size
      const newNumPages = Math.ceil((currentPage * pageSize) / newPageSize);
      // Ensure current page is valid
      setCurrentPage(Math.min(currentPage, newNumPages));
    },
    [currentPage, pageSize],
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
    [currentPage, pageSize, numPages, pageSizeOptions, handlePageSizeChange],
  );

  return (
    <PaginationContext.Provider value={value}>
      {children}
    </PaginationContext.Provider>
  );
}
