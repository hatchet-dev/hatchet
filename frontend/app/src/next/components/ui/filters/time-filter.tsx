import { DateTimePicker } from '@/components/molecules/time-picker/date-time-picker';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { startOfMinute } from 'date-fns';
import {
  TIME_PRESETS,
  useTimeFilters,
} from '@/next/hooks/utils/use-time-filters';

interface TimeFilterProps {
  startField?: string;
  endField?: string;
  className?: string;
}

interface FiltersProps {
  children?: React.ReactNode;
  className?: string;
}

export function TimeFilterGroup({
  className,
  children,
  ...props
}: FiltersProps) {
  return (
    <div
      role="filters"
      aria-label="filters"
      className={cn('flex w-full items-center gap-2 md:gap-6', className)}
      {...props}
    >
      {children}
    </div>
  );
}

export function TogglePause() {
  const { pause, resume, isPaused } = useTimeFilters();

  return (
    <Button
      variant={isPaused ? 'default' : 'outline'}
      size="sm"
      className="h-8 px-2 text-xs"
      onClick={() => {
        if (isPaused) {
          resume();
        } else {
          pause();
        }
      }}
    >
      {isPaused ? 'Resume' : 'Pause'}
    </Button>
  );
}

export function TimeFilter({ className }: TimeFilterProps) {
  const {
    filters,
    setTimeFilter,
    activePreset,
    handleTimeFilterChange,
    handleClearTimeFilters,
  } = useTimeFilters();

  const startDate = filters.startTime
    ? new Date(filters.startTime as string)
    : undefined;
  const endDate = filters.endTime
    ? new Date(filters.endTime as string)
    : undefined;

  const handleDateChange = (date: Date | undefined) => {
    if (date) {
      const roundedDate = startOfMinute(date);
      setTimeFilter({
        startTime: roundedDate.toISOString(),
        endTime: undefined,
      });
    } else {
      handleClearTimeFilters();
    }
  };

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      <div className="flex items-center gap-2">
        {Object.entries(TIME_PRESETS).map(([key]) => (
          <Button
            key={key}
            variant={activePreset === key ? 'default' : 'outline'}
            size="sm"
            className="h-8 px-2 text-xs"
            onClick={() =>
              handleTimeFilterChange(key as keyof typeof TIME_PRESETS)
            }
          >
            {key}
          </Button>
        ))}
      </div>
      <div className="flex items-center gap-4">
        <DateTimePicker
          date={startDate}
          setDate={(date) => handleDateChange(date)}
          label="Start Time"
        />
        <DateTimePicker
          date={endDate}
          setDate={(date) => handleDateChange(date)}
          label="End Time"
        />
      </div>
    </div>
  );
}
