import { TimePicker } from './time-picker';
import { Button } from '@/components/v1/ui/button';
import { Calendar } from '@/components/v1/ui/calendar';
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from '@/components/v1/ui/popover';
import { cn } from '@/lib/utils';
import { CalendarIcon } from '@radix-ui/react-icons';
import { add, format } from 'date-fns';

type DateTimePickerProps = {
  date: Date | undefined;
  setDate: (date: Date | undefined) => void;
  label: string;
  triggerClassName?: string;
};

export function DateTimePicker({
  date,
  setDate,
  label,
  triggerClassName,
}: DateTimePickerProps) {
  /**
   * carry over the current time when a user clicks a new day
   * instead of resetting to 00:00
   */
  const handleSelect = (newDay: Date | undefined) => {
    if (!newDay) {
      return;
    }
    if (!date) {
      setDate(newDay);
      return;
    }
    const diff = newDay.getTime() - date.getTime();
    const diffInDays = diff / (1000 * 60 * 60 * 24);
    const newDateFull = add(date, { days: Math.ceil(diffInDays) });
    setDate(newDateFull);
  };

  return (
    <Popover>
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
          initialFocus
        />
        <div className="border-t border-border p-3">
          <TimePicker setDate={setDate} date={date} />
        </div>
      </PopoverContent>
    </Popover>
  );
}
