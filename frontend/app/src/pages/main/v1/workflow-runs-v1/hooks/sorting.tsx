import { ColumnSort, SortingState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

export const useSorting = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const sorting: SortingState = useMemo(() => {
    const sortParam = searchParams.get('sort');

    if (!sortParam) {
      return [];
    }

    return JSON.parse(sortParam).map(({ desc, id }: ColumnSort) => {
      return { id, desc };
    });
  }, [searchParams]);

  const orderDirection = useMemo(() => {
    return sorting.map((s) => `${s.id}:${s.desc ? 'desc' : 'asc'}`).join(',');
  }, [sorting]);

  const setSorting = useCallback(
    (updaterOrValue: Updater<SortingState>) => {
      const newValues =
        typeof updaterOrValue === 'function'
          ? updaterOrValue(sorting)
          : updaterOrValue;

      setSearchParams((prev) => {
        const newParams = new URLSearchParams(prev);
        newParams.set('sort', JSON.stringify(newValues));
        return newParams;
      });
    },
    [setSearchParams, sorting],
  );

  return {
    sorting,
    orderDirection,
    setSorting,
  };
};
