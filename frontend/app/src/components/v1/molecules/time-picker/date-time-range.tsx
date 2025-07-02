import { CalendarIcon, XMarkIcon } from '@heroicons/react/24/outline';
import { format } from 'date-fns';
import { cn } from '@/lib/utils';
import { Button } from '@/components/v1/ui/button';
import { Calendar } from '@/components/v1/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from '@/components/v1/ui/select';
import { TimePicker } from './time-picker';
import { useState, useEffect } from 'react';
import { Label } from '../../ui/label';

export type TimeWindow = '1h' | '6h' | '1d' | '7d';

type DateTimeRangeProps = {
  startDate?: Date;
  endDate?: Date;
  timeWindow?: TimeWindow;
  isCustomRange?: boolean;
  onTimeWindowChange: (timeWindow: TimeWindow) => void;
  onCustomRangeChange: (startDate?: Date, endDate?: Date) => void;
  onEnableCustomMode: () => void;
  onClearCustomRange: () => void;
};

const timeWindowLabels: Record<TimeWindow, string> = {
  '1h': '1 hour',
  '6h': '6 hours',
  '1d': '1 day',
  '7d': '7 days',
};

const formatDateRange = (
  isCustom: boolean,
  since: Date | undefined,
  until: Date | undefined,
  preset: TimeWindow,
) => {
  if (!isCustom) {
    return timeWindowLabels[preset];
  }

  if (since && until) {
    return `${format(since, 'MMM d, HH:mm')} - ${format(until, 'MMM d, HH:mm')}`;
  }

  if (since) {
    return `After ${format(since, 'MMM d, HH:mm')}`;
  }

  if (until) {
    return `Before ${format(until, 'MMM d, HH:mm')}`;
  }

  return 'Custom range';
};

export function DateTimeRange({
  startDate,
  endDate,
  timeWindow = '1d',
  isCustomRange = false,
  onTimeWindowChange,
  onCustomRangeChange,
  onEnableCustomMode,
  onClearCustomRange,
}: DateTimeRangeProps) {
  const [isCustomPopoverOpen, setIsCustomPopoverOpen] = useState(false);
  const [tempStartDate, setTempStartDate] = useState<Date | undefined>(
    startDate,
  );
  const [tempEndDate, setTempEndDate] = useState<Date | undefined>(endDate);

  console.log('Is custom popover open:', isCustomPopoverOpen);

  useEffect(() => {
    if (isCustomRange) {
      setTempStartDate(startDate);
      setTempEndDate(endDate);
    }
  }, [startDate, endDate, isCustomRange]);

  const handleSelectTimeWindow = (value: TimeWindow | 'custom') => {
    if (value === 'custom') {
      onEnableCustomMode();
      setTempStartDate(startDate);
      setTempEndDate(endDate);
      setIsCustomPopoverOpen(true);
    } else {
      onTimeWindowChange(value);
    }
  };

  const handleApplyCustomRange = () => {
    onCustomRangeChange(tempStartDate, tempEndDate);
    setIsCustomPopoverOpen(false);
  };

  const handleClearCustomRange = () => {
    setTempStartDate(undefined);
    setTempEndDate(undefined);
    onCustomRangeChange(undefined, undefined);
  };

  const handleStartDateSelect = (date: Date | undefined) => {
    setTempStartDate(date);
  };

  const handleEndDateSelect = (date: Date | undefined) => {
    setTempEndDate(date);
  };

  return (
    <div className="flex items-center gap-2">
      <Select
        value={isCustomRange ? 'custom' : timeWindow}
        onValueChange={handleSelectTimeWindow}
      >
        <SelectTrigger className={cn('h-9 min-w-0 max-w-[300px]')}>
          <CalendarIcon className="mr-2 h-4 w-4 flex-shrink-0" />
          <span className="truncate">
            {formatDateRange(isCustomRange, startDate, endDate, timeWindow)}
          </span>
        </SelectTrigger>
        <SelectContent>
          <SelectItem value="1h">1 hour</SelectItem>
          <SelectItem value="6h">6 hours</SelectItem>
          <SelectItem value="1d">1 day</SelectItem>
          <SelectItem value="7d">7 days</SelectItem>
          <SelectItem value="custom">Custom range...</SelectItem>
        </SelectContent>
      </Select>

      {isCustomRange && (
        <Popover
          open={isCustomPopoverOpen}
          onOpenChange={setIsCustomPopoverOpen}
        >
          <PopoverTrigger asChild>
            <Button variant="outline" size="sm" className="h-9">
              Edit Range
            </Button>
          </PopoverTrigger>
          <PopoverContent className="w-auto p-0" align="end">
            <div className="p-4 space-y-4">
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <Label className="text-sm font-medium mb-2 block">
                    Start Date
                  </Label>
                  <div className="space-y-2">
                    <Calendar
                      mode="single"
                      selected={tempStartDate}
                      onSelect={handleStartDateSelect}
                      className="rounded-md border"
                    />
                    {tempStartDate && (
                      <div className="p-2 border-t">
                        <TimePicker
                          setDate={setTempStartDate}
                          date={tempStartDate}
                        />
                      </div>
                    )}
                  </div>
                </div>

                <div>
                  <Label className="text-sm font-medium mb-2 block">
                    End Date
                  </Label>
                  <div className="space-y-2">
                    <Calendar
                      mode="single"
                      selected={tempEndDate}
                      onSelect={handleEndDateSelect}
                      className="rounded-md border"
                    />
                    {tempEndDate && (
                      <div className="p-2 border-t">
                        <TimePicker
                          setDate={setTempEndDate}
                          date={tempEndDate}
                        />
                      </div>
                    )}
                  </div>
                </div>
              </div>

              <div className="flex gap-2 pt-2 border-t">
                <Button
                  size="sm"
                  onClick={handleApplyCustomRange}
                  className="flex-1"
                  disabled={!tempStartDate && !tempEndDate}
                >
                  Apply
                </Button>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={handleClearCustomRange}
                >
                  Clear
                </Button>
              </div>
            </div>
          </PopoverContent>
        </Popover>
      )}

      {isCustomRange && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onClearCustomRange}
          className="h-9 px-2"
          title="Clear custom range"
        >
          <XMarkIcon className="h-4 w-4" />
        </Button>
      )}
    </div>
  );
}
