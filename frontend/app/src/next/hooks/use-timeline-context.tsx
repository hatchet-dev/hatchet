import {
  createContext,
  useContext,
  useState,
  ReactNode,
  useCallback,
  useMemo,
} from 'react';
import { V1WorkflowRun } from '@/next/lib/api';
interface TimelineContextState {
  earliest: number;
  latest: number | undefined;
  updateTimeRange: (item: V1WorkflowRun) => void;
  resetTimeRange: () => void;
  timeRange: number;
}

const TimelineContext = createContext<TimelineContextState | null>(null);

interface TimelineProviderProps {
  children: ReactNode;
}

// Helper to check if a timestamp is valid (not empty or the special "0001-01-01" value)
const isValidTimestamp = (timestamp?: string): boolean => {
  if (!timestamp) {
    return false;
  }

  // Check for the special "0001-01-01" timestamp that represents a null value
  if (timestamp.startsWith('0001-01-01')) {
    return false;
  }

  const date = new Date(timestamp);
  // Check if the date is valid and not too far in the past
  return !isNaN(date.getTime()) && date.getFullYear() > 1970;
};

export function TimelineProvider({ children }: TimelineProviderProps) {
  const [earliest, setEarliest] = useState<number>(Date.now());
  const [latest, setLatest] = useState<number | undefined>(undefined);

  const updateTimeRange = useCallback(
    (item: V1WorkflowRun) => {
      // Always include the current time for active items to ensure timeline is relevant
      const currentTime = Date.now();

      // Collect all valid timestamps
      const times: number[] = [];

      // Add timestamps if they exist and are valid
      if (isValidTimestamp(item.metadata.createdAt)) {
        times.push(new Date(item.metadata.createdAt!).getTime());
      }

      if (isValidTimestamp(item.startedAt)) {
        times.push(new Date(item.startedAt!).getTime());
      }

      if (isValidTimestamp(item.finishedAt)) {
        times.push(new Date(item.finishedAt!).getTime());
      }

      // For active workflows (started but not finished), or pending (created but not started),
      // use the current time to ensure timeline stays updated
      if (
        (isValidTimestamp(item.startedAt) &&
          !isValidTimestamp(item.finishedAt)) ||
        (isValidTimestamp(item.createdAt) && !isValidTimestamp(item.startedAt))
      ) {
        times.push(currentTime);
      }

      // Skip if no valid times
      if (times.length === 0) {
        return;
      }

      const earliestTime = Math.min(...times);
      const latestTime = Math.max(...times);

      // Update earliest time if necessary
      if (earliestTime < earliest) {
        setEarliest(earliestTime);
      }

      // Update latest time if necessary
      if (latest === undefined || latestTime > latest) {
        setLatest(latestTime);
      }
    },
    [earliest, latest, setEarliest, setLatest],
  );

  const timeRange = useMemo(() => {
    if (latest === undefined || earliest === undefined) {
      return 1;
    }
    return latest - earliest;
  }, [latest, earliest]);

  const resetTimeRange = useCallback(() => {
    setEarliest(Date.now());
    setLatest(undefined);
  }, [setEarliest, setLatest]);

  const value = {
    earliest,
    latest,
    updateTimeRange,
    timeRange,
    resetTimeRange,
  };

  return (
    <TimelineContext.Provider value={value}>
      {children}
    </TimelineContext.Provider>
  );
}

export default function useTimeline(): TimelineContextState {
  const context = useContext(TimelineContext);
  if (!context) {
    throw new Error(
      'useTimelineContext must be used within a TimelineProvider',
    );
  }
  return context;
}
