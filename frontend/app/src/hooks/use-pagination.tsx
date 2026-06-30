import { useSearchParams } from '@/lib/router-helpers';
import { PaginationState, Updater } from '@tanstack/react-table';
import { useCallback, useEffect, useMemo, useRef } from 'react';

type PaginationQueryShape = {
  i: number; // index
  s: number; // size
};

type UsePaginationProps = {
  key: string;
  // When this value changes (by deep/serialized equality), the page index resets to 0.
  // Pass the active filter values here so changing a filter doesn't leave the user
  // stranded on a page that no longer has results.
  resetPageOnChange?: unknown;
};

const parsePaginationParam = (searchParams: URLSearchParams, key: string) => {
  const rawPaginationParamValue = searchParams.get(key);

  if (!rawPaginationParamValue) {
    return {
      pageIndex: 0,
      pageSize: 50,
    };
  }

  const parsedPaginationState = JSON.parse(rawPaginationParamValue);

  if (
    !parsedPaginationState ||
    typeof parsedPaginationState !== 'object' ||
    !('i' in parsedPaginationState) ||
    !('s' in parsedPaginationState)
  ) {
    return {
      pageIndex: 0,
      pageSize: 50,
    };
  }

  const { i, s }: PaginationQueryShape = parsedPaginationState;

  return {
    pageIndex: i,
    pageSize: s,
  };
};

export const usePagination = ({
  key,
  resetPageOnChange,
}: UsePaginationProps) => {
  const [searchParams, setSearchParams] = useSearchParams();
  const paramKey = `pagination-${key}`;

  const pagination = useMemo<PaginationState>(
    () => parsePaginationParam(searchParams, paramKey),
    [searchParams, paramKey],
  );

  const resetPageOnChangeKey = JSON.stringify(resetPageOnChange);
  const prevResetPageOnChangeKey = useRef(resetPageOnChangeKey);

  useEffect(() => {
    if (prevResetPageOnChangeKey.current === resetPageOnChangeKey) {
      return;
    }

    prevResetPageOnChangeKey.current = resetPageOnChangeKey;

    if (pagination.pageIndex === 0) {
      return;
    }

    setSearchParams((prev) => {
      const currentPagination = parsePaginationParam(prev, paramKey);

      return {
        ...Object.fromEntries(prev.entries()),
        [paramKey]: {
          i: 0,
          s: currentPagination.pageSize,
        },
      };
    });
  }, [resetPageOnChangeKey, pagination.pageIndex, setSearchParams, paramKey]);

  const offset = useMemo(() => {
    if (!pagination) {
      return 0;
    }

    return pagination.pageIndex * pagination.pageSize;
  }, [pagination]);

  const limit = useMemo(() => {
    if (!pagination) {
      return 25;
    }

    return pagination.pageSize;
  }, [pagination]);

  const nextPage = useCallback(() => {
    setSearchParams((prev) => {
      const currentPagination = parsePaginationParam(prev, paramKey);

      const nextPagination = {
        ...currentPagination,
        pageIndex: currentPagination.pageIndex + 1,
      };

      return {
        ...Object.fromEntries(prev.entries()),
        [paramKey]: {
          i: nextPagination.pageIndex,
          s: nextPagination.pageSize,
        },
      };
    });
  }, [setSearchParams, paramKey]);

  const prevPage = useCallback(() => {
    setSearchParams((prev) => {
      const currentPagination = parsePaginationParam(prev, paramKey);

      const prevPagination = {
        ...currentPagination,
        pageIndex: Math.max(0, currentPagination.pageIndex - 1),
      };

      return {
        ...Object.fromEntries(prev.entries()),
        [paramKey]: {
          i: prevPagination.pageIndex,
          s: prevPagination.pageSize,
        },
      };
    });
  }, [setSearchParams, paramKey]);

  const setPageSize = useCallback(
    (pageSize: number) => {
      setSearchParams((prev) => {
        const currentPagination = parsePaginationParam(prev, paramKey);
        const nextPagination = {
          ...currentPagination,
          pageIndex: 0, // Reset to first page when page size changes
          pageSize,
        };

        return {
          ...Object.fromEntries(prev.entries()),
          [paramKey]: {
            i: nextPagination.pageIndex,
            s: nextPagination.pageSize,
          },
        };
      });
    },
    [setSearchParams, paramKey],
  );

  return {
    pagination,
    setPagination: (updater: Updater<PaginationState>) => {
      setSearchParams((prev) => {
        const currentPagination = parsePaginationParam(prev, paramKey);
        const newPagination =
          typeof updater === 'function' ? updater(currentPagination) : updater;

        return {
          ...Object.fromEntries(prev.entries()),
          [paramKey]: {
            i: newPagination.pageIndex,
            s: newPagination.pageSize,
          },
        };
      });
    },
    offset,
    limit,
    nextPage,
    prevPage,
    setPageSize,
  };
};
