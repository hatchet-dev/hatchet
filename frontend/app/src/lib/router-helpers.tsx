import {
  Outlet as TanStackOutlet,
  useLocation,
  useNavigate as useTanStackNavigate,
  useRouter,
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
      | Record<string, string | number | boolean | null | undefined>
      | ((
          prev: URLSearchParams,
        ) =>
          | URLSearchParams
          | string
          | Record<string, string | number | boolean | null | undefined>),
    options?: SearchNavigateOptions,
  ) => void,
] {
  const location = useLocation();
  const router = useRouter();
  const navigate = useTanStackNavigate();

  const searchParams = useMemo(
    () => new URLSearchParams(location.searchStr ?? ''),
    [location.searchStr],
  );

  const setSearchParams = (
    init:
      | URLSearchParams
      | string
      | Record<string, string | number | boolean | null | undefined>
      | ((
          prev: URLSearchParams,
        ) =>
          | URLSearchParams
          | string
          | Record<string, string | number | boolean | null | undefined>),
    options?: SearchNavigateOptions,
  ) => {
    const prev = new URLSearchParams(location.searchStr ?? '');
    const resolved = typeof init === 'function' ? init(prev) : init;

    let next: URLSearchParams;
    if (resolved instanceof URLSearchParams) {
      next = resolved;
    } else if (typeof resolved === 'string') {
      next = new URLSearchParams(resolved);
    } else {
      next = new URLSearchParams();
      Object.entries(resolved || {}).forEach(([key, value]) => {
        if (value === undefined || value === null) {
          return;
        }
        next.set(key, String(value));
      });
    }

    const searchStr = next.toString();
    navigate({
      to: router.state.location.pathname,
      search: searchStr ? () => Object.fromEntries(next.entries()) : {},
      replace: options?.replace,
      state: options?.state,
    });
  };

  return [searchParams, setSearchParams];
}
