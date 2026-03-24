import { useSearchParams } from '@/lib/router-helpers';
import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  useState,
} from 'react';

const PARAM_KEY = 'rd';

type ParamState = {
  tab?: string;
  task?: string;
  span?: string;
  group?: string;
  q?: string;
};

export type RunDetailSearchState = {
  tab: string | undefined;
  focusedTaskRunId: string | undefined;
  selectedSpanId: string | undefined;
  selectedGroupId: string | undefined;
  queryString: string;
  setTab: (tab: string) => void;
  setFocusedTaskRunId: (id: string | undefined) => void;
  setSelectedSpanId: (id: string | undefined) => void;
  setSelectedGroupId: (id: string | undefined) => void;
  setQueryString: (q: string) => void;
  set: (patch: {
    tab?: string;
    focusedTaskRunId?: string;
    selectedSpanId?: string;
    selectedGroupId?: string;
    queryString?: string;
  }) => void;
};

const RunDetailSearchContext = createContext<RunDetailSearchState | null>(null);

function cleanParams(state: ParamState): ParamState | undefined {
  const cleaned: Record<string, string> = {};
  let hasValue = false;
  for (const [key, value] of Object.entries(state)) {
    if (value !== undefined && value !== null && value !== '') {
      cleaned[key] = value;
      hasValue = true;
    }
  }
  return hasValue ? (cleaned as ParamState) : undefined;
}

function patchToParams(patch: {
  tab?: string;
  focusedTaskRunId?: string;
  selectedSpanId?: string;
  selectedGroupId?: string;
  queryString?: string;
}): Partial<ParamState> {
  const params: Partial<ParamState> = {};
  if ('tab' in patch) {
    params.tab = patch.tab;
  }
  if ('focusedTaskRunId' in patch) {
    params.task = patch.focusedTaskRunId;
  }
  if ('selectedSpanId' in patch) {
    params.span = patch.selectedSpanId;
    if (patch.selectedSpanId) {
      params.group = undefined;
    }
  }
  if ('selectedGroupId' in patch) {
    params.group = patch.selectedGroupId;
    if (patch.selectedGroupId) {
      params.span = undefined;
    }
  }
  if ('queryString' in patch) {
    params.q = patch.queryString || undefined;
  }
  return params;
}

export function RunDetailSearchProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [searchParams, setSearchParams] = useSearchParams();

  const rdParam = searchParams.get(PARAM_KEY);
  const state = useMemo((): ParamState => {
    if (!rdParam) {
      return {};
    }
    try {
      const parsed =
        typeof rdParam === 'string' ? JSON.parse(rdParam) : rdParam;
      return parsed && typeof parsed === 'object' ? parsed : {};
    } catch {
      return {};
    }
  }, [rdParam]);

  const update = useCallback(
    (patch: Partial<ParamState>) => {
      setSearchParams(
        (prev) => {
          const rawCurrent = prev.get(PARAM_KEY);
          let current: ParamState = {};
          if (rawCurrent) {
            try {
              const parsed =
                typeof rawCurrent === 'string'
                  ? JSON.parse(rawCurrent)
                  : rawCurrent;
              current = parsed && typeof parsed === 'object' ? parsed : {};
            } catch {
              /* keep empty */
            }
          }

          const next = cleanParams({ ...current, ...patch });
          const entries: Record<string, unknown> = {};
          prev.forEach((value, key) => {
            if (key !== PARAM_KEY) {
              try {
                entries[key] = JSON.parse(value);
              } catch {
                entries[key] = value;
              }
            }
          });

          if (next) {
            entries[PARAM_KEY] = next;
          }

          return entries;
        },
        { replace: true },
      );
    },
    [setSearchParams],
  );

  const value = useMemo<RunDetailSearchState>(
    () => ({
      tab: state.tab,
      focusedTaskRunId: state.task,
      selectedSpanId: state.span,
      selectedGroupId: state.group,
      queryString: state.q ?? '',
      setTab: (tab: string) => update({ tab }),
      setFocusedTaskRunId: (id: string | undefined) => update({ task: id }),
      setSelectedSpanId: (id: string | undefined) =>
        update({ span: id, group: undefined }),
      setSelectedGroupId: (id: string | undefined) =>
        update({ group: id, span: undefined }),
      setQueryString: (q: string) => update({ q: q || undefined }),
      set: (patch) => update(patchToParams(patch)),
    }),
    [state, update],
  );

  return (
    <RunDetailSearchContext.Provider value={value}>
      {children}
    </RunDetailSearchContext.Provider>
  );
}

export function RunDetailSearchLocalProvider({
  children,
}: {
  children: React.ReactNode;
}) {
  const [tab, setTab] = useState<string | undefined>();
  const [focusedTaskRunId, setFocusedTaskRunId] = useState<
    string | undefined
  >();
  const [selectedSpanId, setSelectedSpanIdRaw] = useState<string | undefined>();
  const [selectedGroupId, setSelectedGroupIdRaw] = useState<
    string | undefined
  >();
  const [queryString, setQueryString] = useState('');

  const value = useMemo<RunDetailSearchState>(
    () => ({
      tab,
      focusedTaskRunId,
      selectedSpanId,
      selectedGroupId,
      queryString,
      setTab,
      setFocusedTaskRunId,
      setSelectedSpanId: (id: string | undefined) => {
        setSelectedSpanIdRaw(id);
        if (id) {
          setSelectedGroupIdRaw(undefined);
        }
      },
      setSelectedGroupId: (id: string | undefined) => {
        setSelectedGroupIdRaw(id);
        if (id) {
          setSelectedSpanIdRaw(undefined);
        }
      },
      setQueryString,
      set: (patch) => {
        if ('tab' in patch) {
          setTab(patch.tab);
        }
        if ('focusedTaskRunId' in patch) {
          setFocusedTaskRunId(patch.focusedTaskRunId);
        }
        if ('selectedSpanId' in patch) {
          setSelectedSpanIdRaw(patch.selectedSpanId);
          if (patch.selectedSpanId) {
            setSelectedGroupIdRaw(undefined);
          }
        }
        if ('selectedGroupId' in patch) {
          setSelectedGroupIdRaw(patch.selectedGroupId);
          if (patch.selectedGroupId) {
            setSelectedSpanIdRaw(undefined);
          }
        }
        if ('queryString' in patch) {
          setQueryString(patch.queryString ?? '');
        }
      },
    }),
    [tab, focusedTaskRunId, selectedSpanId, selectedGroupId, queryString],
  );

  return (
    <RunDetailSearchContext.Provider value={value}>
      {children}
    </RunDetailSearchContext.Provider>
  );
}

export function useRunDetailSearch(): RunDetailSearchState {
  const ctx = useContext(RunDetailSearchContext);
  if (!ctx) {
    throw new Error(
      'useRunDetailSearch must be used within a RunDetailSearchProvider',
    );
  }
  return ctx;
}
