import {
  Outlet as TanStackOutlet,
  useLocation,
  useNavigate as useTanStackNavigate,
} from '@tanstack/react-router';
import type { NavigateOptions } from '@tanstack/react-router';
import { createContext, useContext, useMemo } from 'react';

type SearchNavigateOptions = Pick<NavigateOptions, 'replace' | 'state'>;

const OutletContext = createContext<unknown>(undefined);

export function OutletWithContext({ context }: { context?: unknown }) {
  const parentContext = useContext(OutletContext);
  const value = context === undefined ? parentContext : context;

  return (
    <OutletContext.Provider value={value}>
      <TanStackOutlet />
    </OutletContext.Provider>
  );
}

export function useOutletContext<T>() {
  return useContext(OutletContext) as T;
}

export function useSearchParams(): [
  URLSearchParams,
  (
    init:
      | URLSearchParams
      | string
      | Record<string, unknown>
      | ((
          prev: URLSearchParams,
        ) => URLSearchParams | string | Record<string, unknown>),
    options?: SearchNavigateOptions,
  ) => void,
] {
  const location = useLocation();
  const navigate = useTanStackNavigate();

  const searchParams = useMemo(
    () => new URLSearchParams(location.searchStr ?? ''),
    [location.searchStr],
  );

  const setSearchParams = (
    init:
      | URLSearchParams
      | string
      | Record<string, unknown>
      | ((
          prev: URLSearchParams,
        ) => URLSearchParams | string | Record<string, unknown>),
    options?: SearchNavigateOptions,
  ) => {
    const prev = new URLSearchParams(location.searchStr ?? '');
    const resolved = typeof init === 'function' ? init(prev) : init;

    let searchObject: Record<string, unknown>;
    if (resolved instanceof URLSearchParams) {
      searchObject = {};
      resolved.forEach((value, key) => {
        try {
          searchObject[key] = JSON.parse(value);
        } catch {
          searchObject[key] = value;
        }
      });
    } else if (typeof resolved === 'string') {
      const params = new URLSearchParams(resolved);
      searchObject = {};
      params.forEach((value, key) => {
        try {
          searchObject[key] = JSON.parse(value);
        } catch {
          searchObject[key] = value;
        }
      });
    } else {
      searchObject = {};
      Object.entries(resolved || {}).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          if (typeof value === 'string') {
            try {
              searchObject[key] = JSON.parse(value);
            } catch {
              searchObject[key] = value;
            }
          } else {
            searchObject[key] = value;
          }
        }
      });
    }

    navigate({
      to: location.pathname,
      search: searchObject,
      replace: options?.replace,
      state: options?.state,
    });
  };

  return [searchParams, setSearchParams];
}

export function getOptionalStringParam(
  params: unknown,
  key: string,
): string | undefined {
  if (!params || typeof params !== 'object') {
    return undefined;
  }

  const value = Reflect.get(params, key);
  return typeof value === 'string' ? value : undefined;
}
