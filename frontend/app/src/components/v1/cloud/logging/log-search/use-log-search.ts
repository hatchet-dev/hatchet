import { useState, useCallback, useMemo } from 'react';
import { ParsedLogQuery } from './types';
import { parseLogQuery } from './parser';
import { ListCloudLogsQuery } from '@/lib/api/queries';

export interface UseLogSearchOptions {
  initialQuery?: string;
}

export interface UseLogSearchReturn {
  queryString: string;
  setQueryString: (value: string) => void;
  parsedQuery: ParsedLogQuery;
  apiQueryParams: ListCloudLogsQuery;
  handleQueryChange: (query: ParsedLogQuery) => void;
  clearSearch: () => void;
}

export function useLogSearch(
  options: UseLogSearchOptions = {},
): UseLogSearchReturn {
  const { initialQuery = '' } = options;

  const [queryString, setQueryString] = useState(initialQuery);

  const parsedQuery = useMemo(
    () => parseLogQuery(queryString),
    [queryString],
  );

  const apiQueryParams = useMemo((): ListCloudLogsQuery => {
    const params: ListCloudLogsQuery = {};

    if (parsedQuery.after) {
      params.after = parsedQuery.after;
    }

    if (parsedQuery.before) {
      params.before = parsedQuery.before;
    }

    const searchParts: string[] = [];

    if (parsedQuery.search) {
      searchParts.push(parsedQuery.search);
    }

    if (parsedQuery.level) {
      searchParts.push(`level:${parsedQuery.level}`);
    }

    for (const [key, value] of Object.entries(parsedQuery.metadata)) {
      searchParts.push(`${key}:${value}`);
    }

    if (searchParts.length > 0) {
      params.search = searchParts.join(' ');
    }

    return params;
  }, [parsedQuery]);

  const handleQueryChange = useCallback((_query: ParsedLogQuery) => {
    // This is called when the parsed query updates
    // Can be used for analytics, validation, etc.
  }, []);

  const clearSearch = useCallback(() => {
    setQueryString('');
  }, []);

  return {
    queryString,
    setQueryString,
    parsedQuery,
    apiQueryParams,
    handleQueryChange,
    clearSearch,
  };
}
