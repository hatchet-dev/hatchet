import { Button } from '@/components/v1/ui/button';
import { XCircleIcon } from '@heroicons/react/24/outline';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { TimeWindow } from '../../hooks/use-runs-table-state';
import { useRunsContext } from '../../hooks/runs-provider';
import { useCallback } from 'react';

export const TimeFilter = ({ className }: { className?: string } = {}) => {
  const {
    state,
    filters,
    actions: { updateFilters },
    display: { showDateFilter },
  } = useRunsContext();

  const hasParentFilter = !!state.parentTaskExternalId;

  const handleTimeWindowChange = useCallback(
    (value: TimeWindow | 'custom') => {
      if (value !== 'custom') {
        filters.setTimeWindow(value);
      } else {
        updateFilters({ isCustomTimeRange: true });
      }
    },
    [filters, updateFilters],
  );

  const handleCreatedAfterChange = useCallback(
    (date?: string) => updateFilters({ createdAfter: date }),
    [updateFilters],
  );

  const handleFinishedBeforeChange = useCallback(
    (date?: string) => updateFilters({ finishedBefore: date }),
    [updateFilters],
  );

  const handleClearTimeRange = useCallback(
    () => filters.setCustomTimeRange(null),
    [filters],
  );
  if (!showDateFilter || hasParentFilter) {
    return null;
  }

  return (
    <div
      className={
        className || 'flex flex-row justify-end items-center mb-4 gap-2'
      }
    >
      {state.isCustomTimeRange && [
        <Button
          key="clear"
          onClick={handleClearTimeRange}
          variant="outline"
          size="sm"
          className="text-xs h-9 py-2"
        >
          <XCircleIcon className="h-[18px] w-[18px] mr-2" />
          Clear
        </Button>,
        <DateTimePicker
          key="after"
          label="After"
          date={state.createdAfter ? new Date(state.createdAfter) : undefined}
          setDate={(date) => handleCreatedAfterChange(date?.toISOString())}
        />,
        <DateTimePicker
          key="before"
          label="Before"
          date={
            state.finishedBefore ? new Date(state.finishedBefore) : undefined
          }
          setDate={(date) => handleFinishedBeforeChange(date?.toISOString())}
        />,
      ]}
      <Select
        value={state.isCustomTimeRange ? 'custom' : state.timeWindow}
        onValueChange={handleTimeWindowChange}
      >
        <SelectTrigger className="w-fit">
          <SelectValue id="timerange" placeholder="Choose time range" />
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="1h">1 hour</SelectItem>
          <SelectItem value="6h">6 hours</SelectItem>
          <SelectItem value="1d">1 day</SelectItem>
          <SelectItem value="7d">7 days</SelectItem>
          <SelectItem value="custom">Custom</SelectItem>
        </SelectContent>
      </Select>
    </div>
  );
};
