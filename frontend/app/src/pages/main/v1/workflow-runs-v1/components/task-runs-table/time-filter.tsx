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

interface TimeFilterProps {
  timeWindow: TimeWindow;
  isCustomTimeRange: boolean;
  createdAfter?: string;
  finishedBefore?: string;
  onTimeWindowChange: (timeWindow: TimeWindow | 'custom') => void;
  onCreatedAfterChange: (date?: string) => void;
  onFinishedBeforeChange: (date?: string) => void;
  onClearTimeRange: () => void;
  showDateFilter: boolean;
  hasParentFilter: boolean;
}

export const TimeFilter = ({
  timeWindow,
  isCustomTimeRange,
  createdAfter,
  finishedBefore,
  onTimeWindowChange,
  onCreatedAfterChange,
  onFinishedBeforeChange,
  onClearTimeRange,
  showDateFilter,
  hasParentFilter,
}: TimeFilterProps) => {
  if (!showDateFilter || hasParentFilter) {
    return null;
  }

  return (
    <div className="flex flex-row justify-end items-center mb-4 gap-2">
      {isCustomTimeRange && [
        <Button
          key="clear"
          onClick={onClearTimeRange}
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
          date={createdAfter ? new Date(createdAfter) : undefined}
          setDate={(date) => onCreatedAfterChange(date?.toISOString())}
        />,
        <DateTimePicker
          key="before"
          label="Before"
          date={finishedBefore ? new Date(finishedBefore) : undefined}
          setDate={(date) => onFinishedBeforeChange(date?.toISOString())}
        />,
      ]}
      <Select
        value={isCustomTimeRange ? 'custom' : timeWindow}
        onValueChange={onTimeWindowChange}
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
