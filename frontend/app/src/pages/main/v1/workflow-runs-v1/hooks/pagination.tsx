import { PaginationState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

export const usePagination = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const pagination = useMemo(
    () => ({
      pageIndex: Number(searchParams.get('pageIndex')) || 0,
      pageSize: Number(searchParams.get('pageSize')) || 50,
    }),
    [searchParams],
  );

  const setPagination = useCallback(
    (updaterOrValue: Updater<PaginationState>) => {
      const newValues =
        typeof updaterOrValue === 'function'
          ? updaterOrValue(pagination)
          : updaterOrValue;

      setSearchParams((prev) => {
        const newParams = new URLSearchParams(prev);
        newParams.set('pageSize', String(newValues.pageSize));
        newParams.set('pageIndex', String(newValues.pageIndex));
        return newParams;
      });
    },
    [pagination, setSearchParams],
  );

  const setPageSize = useCallback(
    (newPageSize: number) => {
      setSearchParams((prev) => {
        const newParams = new URLSearchParams(prev);

        newParams.set('pageSize', String(newPageSize));

        return newParams;
      });
    },
    [setSearchParams],
  );

  const offset = useMemo(() => {
    if (!pagination) {
      return;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  return {
    pagination,
    setPagination,
    setPageSize,
    offset,
  };
};
