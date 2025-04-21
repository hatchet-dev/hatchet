import { DateTimePicker } from '@/components/molecules/time-picker/date-time-picker';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { startOfMinute } from 'date-fns';
import { TIME_PRESETS, useTimeFilters } from '@/next/hooks/utils/use-time-filters';

interface TimeFilterProps {
  startField?: string;
  endField?: string;
  className?: string;
}

export function TimeFilter({
  startField = 'createdAfter',
  endField = 'createdBefore',
  className,
}: TimeFilterProps) {
  const { filters, setFilters, activePreset, handleTimeFilterChange } =
    useTimeFilters({
      startField,
      endField,
    });

  const startDate = filters[startField]
    ? new Date(filters[startField] as string)
    : undefined;
  const endDate = filters[endField]
    ? new Date(filters[endField] as string)
    : undefined;

  const handleDateChange = (date: Date | undefined, field: string) => {
    if (date) {
      const roundedDate = startOfMinute(date);
      setFilters({
        ...filters,
        [field]: roundedDate.toISOString(),
      });
      handleTimeFilterChange(null); // Clear preset when manually selecting dates
    } else {
      setFilters({
        ...filters,
        [field]: undefined,
      });
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
          setDate={(date) => handleDateChange(date, startField)}
          label="Start Time"
        />
        <DateTimePicker
          date={endDate}
          setDate={(date) => handleDateChange(date, endField)}
          label="End Time"
        />
      </div>
    </div>
  );
}
