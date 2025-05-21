import { useStateAdapter } from '@/next/lib/utils';
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
  const state = useStateAdapter<T>(initialFilters);

  const filters = state.getValues();

  const setFilter = React.useCallback(
    (key: keyof T, value: T[keyof T]) => {
      state.setValue(key as string, value);
    },
    [state],
  );

  const setFilters = React.useCallback(
    (filters: Partial<T>) => {
      state.setValues(filters);
    },
    [state],
  );

  const clearFilter = React.useCallback(
    (key: keyof T) => {
      state.setValue(key as string, undefined as any);
    },
    [state],
  );

  const clearAllFilters = React.useCallback(() => {
    const currentFilters = state.getValues();
    Object.keys(currentFilters).forEach((key) => {
      state.setValue(key, undefined as any);
    });
  }, [state]);

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
