import {
  runningFilterKey,
  statusKey,
} from '../components/v1/task-runs-columns';
import { useRunsContext } from './runs-provider';
import { V1RunningFilter, V1TaskStatus } from '@/lib/api';
import { useCallback, useMemo } from 'react';

const ALL_STATUSES = Object.values(V1TaskStatus);

// The URL stores "no status filter" as every status selected, so the sidebar
// works on the explicit selection and translates back on write.
export function useRunFilterActions() {
  const { filters } = useRunsContext();
  const { columnFilters, setColumnFilters, apiFilters } = filters;

  const selectedStatuses = useMemo(() => {
    const current = apiFilters.statuses ?? ALL_STATUSES;
    return current.length >= ALL_STATUSES.length ? [] : current;
  }, [apiFilters.statuses]);

  const runningFilter = apiFilters.runningFilter;

  const setFilter = useCallback(
    (id: string, value: unknown) => {
      const isEmpty =
        value === undefined ||
        value === null ||
        value === '' ||
        (Array.isArray(value) && value.length === 0);

      setColumnFilters(
        columnFilters
          .filter((f) => f.id !== id)
          .concat(isEmpty ? [] : [{ id, value }]),
      );
    },
    [columnFilters, setColumnFilters],
  );

  // Status and running sub-filter are written together; setting them one after
  // the other would make the second write overwrite the first.
  const writeStatuses = useCallback(
    (next: V1TaskStatus[], nextRunningFilter?: V1RunningFilter) => {
      const keepRunningFilter =
        next.includes(V1TaskStatus.RUNNING) && !!nextRunningFilter;

      setColumnFilters(
        columnFilters
          .filter((f) => f.id !== statusKey && f.id !== runningFilterKey)
          .concat([
            { id: statusKey, value: next.length > 0 ? next : ALL_STATUSES },
          ])
          .concat(
            keepRunningFilter
              ? [{ id: runningFilterKey, value: nextRunningFilter }]
              : [],
          ),
      );
    },
    [columnFilters, setColumnFilters],
  );

  const toggleStatus = useCallback(
    (status: V1TaskStatus) => {
      const next = selectedStatuses.includes(status)
        ? selectedStatuses.filter((s) => s !== status)
        : [...selectedStatuses, status];

      writeStatuses(
        next,
        status === V1TaskStatus.RUNNING ? undefined : runningFilter,
      );
    },
    [selectedStatuses, runningFilter, writeStatuses],
  );

  const isRunningSubChecked = useCallback(
    (sub: V1RunningFilter) =>
      selectedStatuses.includes(V1TaskStatus.RUNNING) &&
      (!runningFilter || runningFilter === sub),
    [selectedStatuses, runningFilter],
  );

  // No running sub-filter means both sub-options are on, so unchecking one
  // keeps the other, and unchecking the last one drops RUNNING entirely.
  const toggleRunningSub = useCallback(
    (sub: V1RunningFilter, other: V1RunningFilter) => {
      const runningOn = selectedStatuses.includes(V1TaskStatus.RUNNING);

      if (!isRunningSubChecked(sub)) {
        if (!runningOn) {
          writeStatuses([...selectedStatuses, V1TaskStatus.RUNNING], sub);
        } else {
          writeStatuses(selectedStatuses, undefined);
        }
        return;
      }

      if (!runningFilter) {
        writeStatuses(selectedStatuses, other);
      } else {
        writeStatuses(
          selectedStatuses.filter((s) => s !== V1TaskStatus.RUNNING),
          undefined,
        );
      }
    },
    [selectedStatuses, runningFilter, isRunningSubChecked, writeStatuses],
  );

  const clearRunningSub = useCallback(
    () => writeStatuses(selectedStatuses, undefined),
    [selectedStatuses, writeStatuses],
  );

  return {
    selectedStatuses,
    runningFilter,
    setFilter,
    toggleStatus,
    isRunningSubChecked,
    toggleRunningSub,
    clearRunningSub,
  };
}
