import * as React from 'react';
import { startOfMinute, subDays, subHours, subMinutes } from 'date-fns';

export const TIME_PRESETS = {
  '1m': (now: Date) => startOfMinute(subMinutes(now, 1)),
  '10m': (now: Date) => startOfMinute(subMinutes(now, 10)),
  '30m': (now: Date) => startOfMinute(subMinutes(now, 30)),
  '1h': (now: Date) => startOfMinute(subHours(now, 1)),
  '6h': (now: Date) => startOfMinute(subHours(now, 6)),
  '24h': (now: Date) => startOfMinute(subHours(now, 24)),
  '7d': (now: Date) => startOfMinute(subDays(now, 7)),
} as const;

type TimePreset = keyof typeof TIME_PRESETS;

interface TimeFilterOptions {
  startField?: string;
  endField?: string;
}

interface TimeFilterState {
  startTime?: string;
  endTime?: string;
  activePreset: TimePreset | null;
}

interface TimeFilterContextType {
  state: TimeFilterState;
  setTimeFilter: (startTime?: string, endTime?: string) => void;
  setActivePreset: (preset: TimePreset | null) => void;
  clearTimeFilters: () => void;
}

const TimeFilterContext = React.createContext<
  TimeFilterContextType | undefined
>(undefined);

export function useTimeFilters(options: TimeFilterOptions = {}) {
  const { startField = 'createdAfter', endField = 'createdBefore' } = options;
  const context = React.useContext(TimeFilterContext);

  if (!context) {
    throw new Error('useTimeFilters must be used within a TimeFilterProvider');
  }

  const { state, setTimeFilter, setActivePreset, clearTimeFilters } = context;

  const handleTimeFilterChange = React.useCallback(
    (preset: TimePreset | null) => {
      setActivePreset(preset);
    },
    [setActivePreset],
  );

  return {
    activePreset: state.activePreset,
    handleTimeFilterChange,
    handleClearTimeFilters: clearTimeFilters,
    filters: {
      [startField]: state.startTime,
      [endField]: state.endTime,
    },
    setFilters: (filters: Partial<Record<string, string | undefined>>) => {
      setTimeFilter(filters[startField], filters[endField]);
    },
  };
}

interface TimeFilterProviderProps {
  children: React.ReactNode;
  options?: TimeFilterOptions;
}

export function TimeFilterProvider({ children }: TimeFilterProviderProps) {
  const [state, setState] = React.useState<TimeFilterState>({
    activePreset: '1h',
  });
  const updateIntervalRef = React.useRef<NodeJS.Timeout>();

  const setDefaultTimeRange = React.useCallback(() => {
    const now = startOfMinute(new Date());
    const startTime = TIME_PRESETS['1h'](now);
    setState((prev) => ({
      ...prev,
      startTime: startTime.toISOString(),
      endTime: undefined,
    }));
  }, []);

  // Set initial 1h time range
  React.useEffect(() => {
    setDefaultTimeRange();
  }, [setDefaultTimeRange]);

  // Clear interval when filters change
  const clearUpdateInterval = React.useCallback(() => {
    if (updateIntervalRef.current) {
      if (typeof updateIntervalRef.current === 'number') {
        clearInterval(updateIntervalRef.current);
      } else {
        clearTimeout(updateIntervalRef.current);
      }
      updateIntervalRef.current = undefined;
    }
  }, []);

  // Handle preset updates
  React.useEffect(() => {
    if (!state.activePreset) {
      clearUpdateInterval();
      return;
    }

    const updateTimeRange = () => {
      const now = startOfMinute(new Date());
      const startTime = TIME_PRESETS[state.activePreset!](now);

      setState((prev) => ({
        ...prev,
        startTime: startTime.toISOString(),
        endTime: undefined,
      }));
    };

    // Update immediately
    updateTimeRange();

    // Set up interval for updates
    const updateEveryMinute = () => {
      const now = new Date();
      const secondsUntilNextMinute = 60 - now.getSeconds();
      const timeoutId = setTimeout(() => {
        updateTimeRange();
        // Once we're synchronized with the minute boundary, use setInterval
        updateIntervalRef.current = setInterval(updateTimeRange, 60 * 1000); // Update every minute
      }, secondsUntilNextMinute * 1000);

      return timeoutId;
    };
    const timeoutId = updateEveryMinute();
    updateIntervalRef.current = timeoutId;

    return clearUpdateInterval;
  }, [state.activePreset, clearUpdateInterval]);

  const setTimeFilter = React.useCallback(
    (startTime?: string, endTime?: string) => {
      setState((prev) => ({
        ...prev,
        startTime,
        endTime,
      }));
    },
    [],
  );

  const setActivePreset = React.useCallback(
    (preset: TimePreset | null) => {
      clearUpdateInterval();
      setState((prev) => ({
        ...prev,
        activePreset: preset,
      }));
    },
    [clearUpdateInterval],
  );

  const clearTimeFilters = React.useCallback(() => {
    clearUpdateInterval();
    setState((prev) => ({
      ...prev,
      activePreset: '1h',
    }));
    setDefaultTimeRange();
  }, [clearUpdateInterval, setDefaultTimeRange]);

  const value = React.useMemo(
    () => ({
      state,
      setTimeFilter,
      setActivePreset,
      clearTimeFilters,
    }),
    [state, setTimeFilter, setActivePreset, clearTimeFilters],
  );

  return (
    <TimeFilterContext.Provider value={value}>
      {children}
    </TimeFilterContext.Provider>
  );
}
