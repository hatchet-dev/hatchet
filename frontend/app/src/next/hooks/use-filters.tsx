import * as React from 'react';

export interface FilterManager<T> {
  filters: T;
  setFilter: (key: keyof T, value: T[keyof T]) => void;
  clearFilter: (key: keyof T) => void;
  clearAllFilters: () => void;
}

export const FilterManagerNoOp: FilterManager<Record<string, any>> = {
  filters: {},
  setFilter: () => {},
  clearFilter: () => {},
  clearAllFilters: () => {},
};

type FilterContextType<T> = FilterManager<T>;

const FilterContext = React.createContext<FilterContextType<any> | undefined>(
  undefined,
);

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

export function FilterProvider<T extends Record<string, any>>({
  children,
  initialFilters = {} as T,
}: FilterProviderProps<T>) {
  const [filters, setFilters] = React.useState<T>(initialFilters);

  const setFilter = React.useCallback((key: keyof T, value: T[keyof T]) => {
    setFilters((prev) => ({
      ...prev,
      [key]: value,
    }));
  }, []);

  const clearFilter = React.useCallback((key: keyof T) => {
    setFilters((prev) => {
      const newFilters = { ...prev };
      delete newFilters[key];
      return newFilters;
    });
  }, []);

  const clearAllFilters = React.useCallback(() => {
    setFilters({} as T);
  }, []);

  const value = React.useMemo(
    () => ({
      filters,
      setFilter,
      clearFilter,
      clearAllFilters,
    }),
    [filters, setFilter, clearFilter, clearAllFilters],
  ) as FilterManager<T>;

  return (
    <FilterContext.Provider value={value}>{children}</FilterContext.Provider>
  );
}
