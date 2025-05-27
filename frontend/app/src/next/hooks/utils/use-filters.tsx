import * as React from 'react';

interface FilterManager<T> {
  filters: T;
  setFilter: (key: keyof T, value: T[keyof T]) => void;
  setFilters: (filters: Partial<T>) => void;
  clearFilter: (key: keyof T) => void;
  clearAllFilters: () => void;
}

type FilterContextType<T> = FilterManager<T>;

const FilterContext = React.createContext<FilterContextType<any> | undefined>(
  undefined,
);

/**
 * Hook to access the current filter state and filter management functions
 * @returns FilterManager instance with filter management functions
 */
export function useFilters<T>() {
  const context = React.useContext<FilterContextType<T> | undefined>(
    FilterContext,
  );

  if (!context) {
    throw new Error('useFilters must be used within a FilterProvider');
  }

  return context as FilterManager<T>;
}

interface FilterProviderProps<T extends Record<string, any>> {
  children: React.ReactNode;
  initialFilters?: T;
}

/**
 * Provider component that manages filter state
 * @param props.children - React children
 * @param props.initialFilters - Initial filter values
 * @param props.type - Storage type ('query' or 'state')
 */
export function FilterProvider<T extends Record<string, any>>({
  children,
  initialFilters = {} as T,
}: FilterProviderProps<T>) {
  const [filters, setFilters] = React.useState<T>(initialFilters);

  const setFilter = React.useCallback(
    (key: keyof T, value: T[keyof T]) => {
      setFilters((prev) => ({
        ...prev,
        [key]: value,
      }));
    },
    [filters],
  );
  const clearFilter = React.useCallback(
    (key: keyof T) => {
      setFilters((prev) => ({
        ...prev,
        [key]: undefined as any,
      }));
    },
    [filters],
  );

  const clearAllFilters = React.useCallback(() => {
    const clearedFilters = Object.keys(filters).reduce((acc, key) => {
      acc[key as keyof T] = undefined as any;
      return acc;
    }, {} as T);
    setFilters(clearedFilters);
  }, [filters]);

  const value = React.useMemo(
    () => ({
      filters,
      setFilter,
      setFilters,
      clearFilter,
      clearAllFilters,
    }),
    [filters, setFilter, setFilters, clearFilter, clearAllFilters],
  ) as FilterManager<T>;

  return (
    <FilterContext.Provider value={value}>{children}</FilterContext.Provider>
  );
}
