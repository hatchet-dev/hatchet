import { TimePicker } from './time-picker';
import { Button } from '@/components/v1/ui/button';
import { Calendar } from '@/components/v1/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { cn } from '@/lib/utils';
import {
  formatRetentionPeriod,
  getRetentionBoundary,
  isBeforeRetention,
} from '@/lib/utils/retention';
import { CalendarIcon } from '@radix-ui/react-icons';
import { add, format } from 'date-fns';
import { useState } from 'react';

type DateTimePickerProps = {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  label: string;
  triggerClassName?: string;
  retentionPeriod?: string;
  onBlockedDate?: (date: Date) => void;
};

export function DateTimePicker({
  date,
  setDate,
  label,
  triggerClassName,
  retentionPeriod,
  onBlockedDate,
}: DateTimePickerProps) {
  const [open, setOpen] = useState(false);
  const fromDate = retentionPeriod
    ? getRetentionBoundary(retentionPeriod) ?? undefined
    : undefined;

  const blockDate = (next: Date) => {
    setOpen(false);
    onBlockedDate?.(next);
  };

  const applyDate = (next: Date | undefined) => {
    if (
      next &&
      retentionPeriod &&
      isBeforeRetention(next, retentionPeriod)
    ) {
      blockDate(next);
      return;
    }
    setDate(next);
  };

  /**
   * carry over the current time when a user clicks a new day
   * instead of resetting to 00:00
   */
  const handleSelect = (newDay: Date | undefined) => {
    if (!newDay) {
      return;
    }
    if (!date) {
      applyDate(newDay);
      return;
    }
    const diff = newDay.getTime() - date.getTime();
    const diffInDays = diff / (1000 * 60 * 60 * 24);
    const newDateFull = add(date, { days: Math.ceil(diffInDays) });
    applyDate(newDateFull);
  };

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button
          variant={'outline'}
          className={cn(
            'w-fit justify-start text-left text-xs font-normal',
            !date && 'text-muted-foreground',
            triggerClassName,
          )}
        >
          <CalendarIcon className="mr-2 size-4" />
          {date ? (
            label + ':  ' + format(date, 'PPP HH:mm:ss')
          ) : (
            <span>{label}</span>
          )}
        </Button>
      </PopoverTrigger>
      <PopoverContent className="w-auto p-0">
        <Calendar
          mode="single"
          selected={date}
          onSelect={(d) => handleSelect(d)}
          modifiers={
            fromDate ? { outsideRetention: { before: fromDate } } : undefined
          }
          modifiersClassNames={
            fromDate
              ? { outsideRetention: 'text-muted-foreground opacity-50' }
              : undefined
          }
          onDayClick={(day, modifiers) => {
            if (modifiers.outsideRetention) {
              blockDate(day);
            }
          }}
          initialFocus
        />
        <div className="border-t border-border p-3">
          <TimePicker setDate={applyDate} date={date} />
        </div>
        {fromDate && retentionPeriod ? (
          <div className="border-t border-border px-3 py-2 text-xs text-muted-foreground">
            Dates before {format(fromDate, 'MMM d')} are outside your{' '}
            {formatRetentionPeriod(retentionPeriod)} retention.
          </div>
        ) : null}
      </PopoverContent>
    </Popover>
  );
}
