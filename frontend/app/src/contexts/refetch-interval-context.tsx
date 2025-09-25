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
  isFrozen: boolean;
  setIsFrozen: (isFrozen: boolean) => void;
  userRefetchIntervalPreference: LabeledRefetchInterval;
  refetchInterval: number | false;
  setRefetchInterval: (interval: LabeledRefetchInterval) => void;
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
    useLocalStorageState<RefetchIntervalOption>(STORAGE_KEY, '10s');
  const [isFrozen, setIsFrozen] = useLocalStorageState<boolean>(
    'app-refetch-interval-frozen',
    false,
  );

  const userRefetchIntervalPreference = useMemo(
    () => RefetchInterval[storedInterval],
    [storedInterval],
  );

  const refetchInterval = useMemo(() => {
    if (isFrozen) {
      return false;
    }

    return userRefetchIntervalPreference.value;
  }, [isFrozen, userRefetchIntervalPreference]);

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

  const value = useMemo<RefetchIntervalContextType>(
    () => ({
      isFrozen,
      setIsFrozen,
      userRefetchIntervalPreference,
      refetchInterval,
      setRefetchInterval,
    }),
    [
      refetchInterval,
      setRefetchInterval,
      isFrozen,
      setIsFrozen,
      userRefetchIntervalPreference,
    ],
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
