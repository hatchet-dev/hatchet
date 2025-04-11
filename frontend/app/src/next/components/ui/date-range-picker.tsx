import * as React from 'react';
import { format } from 'date-fns';
import { Calendar as CalendarIcon } from 'lucide-react';
import { DateRange } from 'react-day-picker';

import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { Calendar } from '@/next/components/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/next/components/ui/popover';
import { TimePicker } from '@/next/components/ui/time-picker';

interface DateRangePickerProps {
  value?: DateRange;
  onChange: (value: DateRange | undefined) => void;
  timezone?: string;
  placeholder?: string;
}

export function DateRangePicker({
  value,
  onChange,
  timezone = Intl.DateTimeFormat().resolvedOptions().timeZone,
  placeholder = 'Select date range',
}: DateRangePickerProps) {
  const [, setStartTime] = React.useState<Date | undefined>(value?.from);
  const [, setEndTime] = React.useState<Date | undefined>(value?.to);

  // Update time values when the date range changes
  React.useEffect(() => {
    setStartTime(value?.from);
    setEndTime(value?.to);
  }, [value?.from, value?.to]);

  // Handle start time change
  const handleStartTimeChange = (date: Date | undefined) => {
    if (!date || !value?.from) {
      return;
    }

    const newFrom = new Date(value.from);
    newFrom.setHours(date.getHours());
    newFrom.setMinutes(date.getMinutes());

    onChange({ ...value, from: newFrom });
  };

  // Handle end time change
  const handleEndTimeChange = (date: Date | undefined) => {
    if (!date || !value?.to) {
      return;
    }

    const newTo = new Date(value.to);
    newTo.setHours(date.getHours());
    newTo.setMinutes(date.getMinutes());

    onChange({ ...value, to: newTo });
  };

  return (
    <Popover>
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          className={cn(
            'w-full justify-start text-left',
            !value && 'text-muted-foreground',
          )}
        >
          <CalendarIcon className="mr-2 h-4 w-4" />
          {value?.from ? (
            value.to ? (
              <>
                {format(value.from, 'LLL dd, y HH:mm')} -{' '}
                {format(value.to, 'LLL dd, y HH:mm')}
              </>
            ) : (
              format(value.from, 'LLL dd, y HH:mm')
            )
          ) : (
            placeholder
          )}
          {timezone && (
            <span className="ml-1 text-xs text-muted-foreground">
              ({timezone})
            </span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0" align="start">
        <div className="p-3 space-y-3">
          <div className="grid gap-2">
            <Calendar
              initialFocus
              mode="range"
              defaultMonth={value?.from}
              selected={value}
              onSelect={onChange}
              numberOfMonths={2}
            />
          </div>
          <div className="grid grid-cols-2 gap-4 border-t pt-3">
            <div className="space-y-1">
              <div className="text-sm font-medium">Start Time</div>
              <TimePicker date={value?.from} setDate={handleStartTimeChange} />
            </div>
            <div className="space-y-1">
              <div className="text-sm font-medium">End Time</div>
              <TimePicker date={value?.to} setDate={handleEndTimeChange} />
            </div>
          </div>
        </div>
      </PopoverContent>
    </Popover>
  );
}
