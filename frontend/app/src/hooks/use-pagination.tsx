import { useSearchParams } from '@/lib/router-helpers';
import { PaginationState, Updater } from '@tanstack/react-table';
import { DependencyList, useCallback, useEffect, useMemo, useRef } from 'react';

type PaginationQueryShape = {
  i: number; // index
  s: number; // size
};

type UsePaginationProps = {
  key: string;
  // if any dependency changes, we reset pagination to 0 so users don't get
  // stuck on a page that no longer has results.
  resetPageOnChange?: DependencyList;
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

// Builds the updated search params for setting the `i` (pageIndex) and `s`
// (pageSize) of the pagination param at `paramKey`, preserving all other params.
const buildPaginationSearchParams = (
  prev: URLSearchParams,
  paramKey: string,
  i: number,
  s: number,
) => ({
  ...Object.fromEntries(prev.entries()),
  [paramKey]: { i, s },
});

// JSON.stringify throws on values a DependencyList can legally contain but
// JSON can't represent (BigInt, circular references). Fall back to a
// best-effort string so a bad dependency value can't crash render.
const serializeDeps = (deps?: DependencyList): string => {
  if (!deps) {
    return '';
  }

  try {
    return JSON.stringify(deps);
  } catch {
    return deps.map((dep) => String(dep)).join('|');
  }
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

  const resetPageOnChangeKey = serializeDeps(resetPageOnChange);
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

      return buildPaginationSearchParams(
        prev,
        paramKey,
        0,
        currentPagination.pageSize,
      );
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

      return buildPaginationSearchParams(
        prev,
        paramKey,
        currentPagination.pageIndex + 1,
        currentPagination.pageSize,
      );
    });
  }, [setSearchParams, paramKey]);

  const prevPage = useCallback(() => {
    setSearchParams((prev) => {
      const currentPagination = parsePaginationParam(prev, paramKey);

      return buildPaginationSearchParams(
        prev,
        paramKey,
        Math.max(0, currentPagination.pageIndex - 1),
        currentPagination.pageSize,
      );
    });
  }, [setSearchParams, paramKey]);

  const setPageSize = useCallback(
    (pageSize: number) => {
      setSearchParams((prev) =>
        // Reset to first page when page size changes
        buildPaginationSearchParams(prev, paramKey, 0, pageSize),
      );
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

        return buildPaginationSearchParams(
          prev,
          paramKey,
          newPagination.pageIndex,
          newPagination.pageSize,
        );
      });
    },
    offset,
    limit,
    nextPage,
    prevPage,
    setPageSize,
  };
};
