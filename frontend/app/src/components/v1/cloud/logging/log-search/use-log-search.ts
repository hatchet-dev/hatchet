import { parseLogQuery } from './parser';
import { ParsedLogQuery } from './types';
import { ListCloudLogsQuery } from '@/lib/api/queries';
import { useState, useCallback, useMemo } from 'react';

export interface UseLogSearchReturn {
  queryString: string;
  setQueryString: (value: string) => void;
  parsedQuery: ParsedLogQuery;
  apiQueryParams: ListCloudLogsQuery;
  clearSearch: () => void;
}

export function useLogSearch(initialQuery = ''): UseLogSearchReturn {
  const [queryString, setQueryString] = useState(initialQuery);

  const parsedQuery = useMemo(() => parseLogQuery(queryString), [queryString]);

  const apiQueryParams = useMemo((): ListCloudLogsQuery => {
    const params: ListCloudLogsQuery = {};
    const searchParts: string[] = [];

    if (parsedQuery.search) {
      searchParts.push(parsedQuery.search);
    }

    if (parsedQuery.level) {
      searchParts.push(`level:${parsedQuery.level}`);
    }

    if (searchParts.length > 0) {
      params.search = searchParts.join(' ');
    }

    return params;
  }, [parsedQuery]);

  const clearSearch = useCallback(() => {
    setQueryString('');
  }, []);

  return {
    queryString,
    setQueryString,
    parsedQuery,
    apiQueryParams,
    clearSearch,
  };
}
