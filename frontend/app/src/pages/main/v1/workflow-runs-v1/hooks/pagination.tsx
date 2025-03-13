import { PaginationState, Updater } from '@tanstack/react-table';
import { useCallback, useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';

export const usePagination = () => {
  const [searchParams, setSearchParams] = useSearchParams();

  const pageSizeParamName = 'pageSize';
  const pageIndexParamName = 'pageIndex';

  const pagination = useMemo(
    () => ({
      pageIndex: Number(searchParams.get(pageSizeParamName)) || 0,
      pageSize: Number(searchParams.get(pageIndexParamName)) || 50,
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
        newParams.set(pageSizeParamName, String(newValues.pageSize));
        newParams.set(pageIndexParamName, String(newValues.pageIndex));
        return newParams;
      });
    },
    [pagination, setSearchParams],
  );

  const setPageSize = useCallback(
    (newPageSize: number) => {
      setSearchParams((prev) => {
        const newParams = new URLSearchParams(prev);

        newParams.set(pageSizeParamName, String(newPageSize));

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
    pageIndexParamName,
  };
};
