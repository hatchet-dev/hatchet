import * as React from 'react';
import {
  endOfMinute,
  startOfMinute,
  subDays,
  subHours,
  subMinutes,
} from 'date-fns';

export const TIME_PRESETS = {
  '30m': (now: Date) => startOfMinute(subMinutes(now, 30)),
  '1h': (now: Date) => startOfMinute(subHours(now, 1)),
  '6h': (now: Date) => startOfMinute(subHours(now, 6)),
  '24h': (now: Date) => startOfMinute(subHours(now, 24)),
  '7d': (now: Date) => startOfMinute(subDays(now, 7)),
} as const;

type TimePreset = keyof typeof TIME_PRESETS;

interface CustomTimeRange {
  startTime: string;
  endTime?: string;
}

type TimeFilterInput = TimePreset | CustomTimeRange;

interface TimeFilterOptions {
  startField?: string;
  endField?: string;
}

interface TimeFilterState {
  startTime?: string;
  endTime?: string;
  activePreset: TimePreset | null;
  lastActivePreset: TimePreset | null;
}

interface TimeFilterContextType {
  state: TimeFilterState;
  setTimeFilter: (input: TimeFilterInput) => void;
  setActivePreset: (preset: TimePreset | null) => void;
  clearTimeFilters: () => void;
}

const TimeFilterContext = React.createContext<
  TimeFilterContextType | undefined
>(undefined);

export function useTimeFilters() {
  const context = React.useContext(TimeFilterContext);

  if (!context) {
    throw new Error('useTimeFilters must be used within a TimeFilterProvider');
  }

  const { state, setTimeFilter, setActivePreset, clearTimeFilters } = context;

  const handleTimeFilterChange = React.useCallback(
    (preset: TimePreset | null) => {
      if (preset) {
        setTimeFilter(preset);
      } else {
        setActivePreset(null);
      }
    },
    [setTimeFilter, setActivePreset],
  );

  return {
    activePreset: state.activePreset,
    handleTimeFilterChange,
    handleClearTimeFilters: clearTimeFilters,
    filters: {
      startTime: state.startTime,
      endTime: state.endTime,
    },
    setTimeFilter,
    pause: () => {
      setTimeFilter({
        startTime: state.startTime!,
        endTime: endOfMinute(new Date()).toISOString(),
      });
    },
    resume: () => {
      if (state.lastActivePreset) {
        setTimeFilter(state.lastActivePreset);
      } else {
        setTimeFilter({
          startTime: state.startTime!,
          endTime: undefined,
        });
      }
    },
    isPaused: state.endTime !== undefined,
  };
}

interface TimeFilterProviderProps {
  children: React.ReactNode;
  options?: TimeFilterOptions;
  initialTimeRange?: {
    startTime?: string;
    endTime?: string;
    activePreset?: keyof typeof TIME_PRESETS;
  };
}

export function TimeFilterProvider({
  children,
  initialTimeRange,
}: TimeFilterProviderProps) {
  const [state, setState] = React.useState<TimeFilterState>({
    activePreset: initialTimeRange?.activePreset || '1h',
    lastActivePreset: initialTimeRange?.activePreset || '1h',
    startTime: initialTimeRange?.startTime,
    endTime: initialTimeRange?.endTime,
  });

  const setTimeFilter = React.useCallback((input: TimeFilterInput) => {
    if (typeof input === 'string') {
      const now = startOfMinute(new Date());
      const startTime = TIME_PRESETS[input](now);
      setState((prev) => ({
        ...prev,
        startTime: startTime.toISOString(),
        endTime: undefined,
        activePreset: input,
        lastActivePreset: input,
      }));
    } else {
      setState((prev) => ({
        ...prev,
        startTime: input.startTime,
        endTime: input.endTime,
        activePreset: null,
      }));
    }
  }, []);

  const setActivePreset = React.useCallback((preset: TimePreset | null) => {
    setState((prev) => ({
      ...prev,
      activePreset: preset,
    }));
  }, []);

  const clearTimeFilters = React.useCallback(() => {
    setState((prev) => ({
      ...prev,
      activePreset: '1h',
    }));
  }, []);

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
