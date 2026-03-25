import { useSearchParams } from '@/lib/router-helpers';
import { ColumnFiltersState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { z } from 'zod';

type FilterMapping<T> = {
  [K in keyof T]: string;
};

export function useZodColumnFilters<T extends z.ZodType>(
  schema: T,
  key: string,
  filterMapping: FilterMapping<z.infer<T>>,
): {
  state: z.infer<T>;
  columnFilters: ColumnFiltersState;
  setColumnFilters: (updater: Updater<ColumnFiltersState>) => void;
  resetFilters: () => void;
} {
  const [searchParams, setSearchParams] = useSearchParams();

  const state = useMemo((): z.infer<T> => {
    const rawValue = searchParams.get(key);

    if (!rawValue) {
      return schema.parse({});
    }

    try {
      const parsed =
        typeof rawValue === 'string' ? JSON.parse(rawValue) : rawValue;
      const validated = schema.parse(parsed);
      return validated;
    } catch (e) {
      return schema.parse({});
    }
  }, [searchParams, key, schema]);

  const setQueryState = useCallback(
    (newValue: z.infer<T>) => {
      setSearchParams((prev) => ({
        ...Object.fromEntries(prev.entries()),
        [key]: newValue,
      }));
    },
    [key, setSearchParams],
  );

  const columnFilters = useMemo<ColumnFiltersState>(() => {
    const filters: ColumnFiltersState = [];

    Object.entries(filterMapping).forEach(([schemaKey, columnId]) => {
      const value = state[schemaKey as keyof typeof state];
      if (value !== undefined && value !== null) {
        if (Array.isArray(value) && value.length > 0) {
          filters.push({ id: columnId, value });
        } else if (typeof value === 'string' && value.length > 0) {
          filters.push({ id: columnId, value });
        } else if (typeof value === 'number' || typeof value === 'boolean') {
          filters.push({ id: columnId, value });
        }
      }
    });

    return filters;
  }, [state, filterMapping]);

  const setColumnFilters = useCallback(
    (updater: Updater<ColumnFiltersState>) => {
      const currentColumnFilters = columnFilters;
      const newColumnFilters =
        typeof updater === 'function' ? updater(currentColumnFilters) : updater;

      const newQueryState = { ...state };

      Object.keys(filterMapping).forEach((schemaKey) => {
        const currentValue = state[schemaKey as keyof typeof state];
        if (Array.isArray(currentValue)) {
          newQueryState[schemaKey] = [];
        } else {
          newQueryState[schemaKey] = undefined;
        }
      });

      newColumnFilters.forEach((filter) => {
        const schemaKey = Object.entries(filterMapping).find(
          ([, columnId]) => columnId === filter.id,
        )?.[0];

        if (schemaKey) {
          newQueryState[schemaKey] = filter.value;
        }
      });

      setQueryState(newQueryState);
    },
    [columnFilters, state, setQueryState, filterMapping],
  );

  const resetFilters = useCallback(() => {
    const defaultState = schema.parse({});
    setQueryState(defaultState);
  }, [schema, setQueryState]);

  return {
    state,
    columnFilters,
    setColumnFilters,
    resetFilters,
  };
}
