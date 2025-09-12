import {
  createContext,
  useCallback,
  useContext,
  useMemo,
  ReactNode,
} from 'react';
import {
  RefetchInterval,
  RefetchIntervalOption,
  LabeledRefetchInterval,
} from '@/lib/api/api';
import { useLocalStorageState } from '@/hooks/use-local-storage-state';

interface RefetchIntervalContextType {
  currentInterval: LabeledRefetchInterval;
  setRefetchInterval: (interval: LabeledRefetchInterval) => void;
  isOff: boolean;
}

const RefetchIntervalContext = createContext<RefetchIntervalContextType | null>(
  null,
);

interface RefetchIntervalProviderProps {
  children: ReactNode;
}

const STORAGE_KEY = 'app-refetch-interval';

export const RefetchIntervalProvider = ({
  children,
}: RefetchIntervalProviderProps) => {
  const [storedInterval, setStoredInterval] =
    useLocalStorageState<RefetchIntervalOption>(STORAGE_KEY, 'off');

  const currentInterval = useMemo(
    () => RefetchInterval[storedInterval],
    [storedInterval],
  );

  const setRefetchInterval = useCallback(
    (interval: LabeledRefetchInterval) => {
      const key = Object.entries(RefetchInterval).find(
        ([, value]) => value.value === interval.value,
      )?.[0] as RefetchIntervalOption;

      if (key) {
        setStoredInterval(key);
      }
    },
    [setStoredInterval],
  );

  const isOff = currentInterval.value === false;

  const value = useMemo<RefetchIntervalContextType>(
    () => ({
      currentInterval,
      setRefetchInterval,
      isOff,
    }),
    [currentInterval, setRefetchInterval, isOff],
  );

  return (
    <RefetchIntervalContext.Provider value={value}>
      {children}
    </RefetchIntervalContext.Provider>
  );
};

export const useRefetchInterval = () => {
  const context = useContext(RefetchIntervalContext);

  if (!context) {
    throw new Error(
      'useRefetchInterval must be used within a RefetchIntervalProvider',
    );
  }

  return context;
};
