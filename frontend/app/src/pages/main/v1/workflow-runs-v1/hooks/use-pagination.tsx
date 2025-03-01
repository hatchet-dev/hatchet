import { PaginationState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

export const usePagination = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const pagination: PaginationState = useMemo(
    () => ({
      pageIndex: Number(searchParams.get('pageIndex')) || 0,
      pageSize: Number(searchParams.get('pageSize')) || 50,
    }),
    [searchParams],
  );

  const setPagination = useCallback(
    (updaterOrValue: Updater<PaginationState>) => {
      if (typeof updaterOrValue === 'function') {
        const newValues = updaterOrValue(pagination);

        setSearchParams((prev) => {
          const newParams = new URLSearchParams(prev);

          newParams.set('pageSize', String(newValues.pageSize));
          newParams.set('pageIndex', String(newValues.pageIndex));

          return newParams;
        });
      } else {
        setSearchParams((prev) => {
          const newParams = new URLSearchParams(prev);

          newParams.set('pageSize', String(updaterOrValue.pageSize));
          newParams.set('pageIndex', String(updaterOrValue.pageIndex));

          return newParams;
        });
      }
    },
    [setSearchParams],
  );

  const setPageSize = useCallback(
    (newPageSize: number) => {
      setPagination({
        ...pagination,
        pageSize: newPageSize,
      });
    },
    [setPagination],
  );

  return {
    pagination,
    setPagination,
    setPageSize,
  };
};
