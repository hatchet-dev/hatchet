import { useFilters } from '@/next/hooks/use-filters';
import { DateTimePicker } from '@/components/molecules/time-picker/date-time-picker';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { subMinutes, subHours, subDays } from 'date-fns';

interface TimeFilterProps<T extends Record<string, any>> {
  startField: keyof T;
  endField: keyof T;
  className?: string;
}

const TIME_PRESETS = [
  { label: '10m', getStartDate: () => subMinutes(new Date(), 10) },
  { label: '30m', getStartDate: () => subMinutes(new Date(), 30) },
  { label: '1h', getStartDate: () => subHours(new Date(), 1) },
  { label: '6h', getStartDate: () => subHours(new Date(), 6) },
  { label: '24h', getStartDate: () => subHours(new Date(), 24) },
  { label: '7d', getStartDate: () => subDays(new Date(), 7) },
] as const;

export function TimeFilter<T extends Record<string, any>>({
  startField,
  endField,
  className,
}: TimeFilterProps<T>) {
  const { filters, setFilters } = useFilters<T>();

  const startDate = filters[startField]
    ? new Date(filters[startField] as string)
    : undefined;
  const endDate = filters[endField]
    ? new Date(filters[endField] as string)
    : undefined;

  const handlePresetClick = (getStartDate: () => Date) => {
    const newStartDate = getStartDate();
    setFilters({
      [startField]: newStartDate.toISOString(),
      [endField]: undefined,
    } as Partial<T>);
  };

  return (
    <div className={cn('flex flex-col gap-2', className)}>
      <div className="flex items-center gap-2">
        {TIME_PRESETS.map((preset) => (
          <Button
            key={preset.label}
            variant="outline"
            size="sm"
            className="h-8 px-2 text-xs"
            onClick={() => handlePresetClick(preset.getStartDate)}
          >
            {preset.label}
          </Button>
        ))}
      </div>
      <div className="flex items-center gap-4">
        <DateTimePicker
          date={startDate}
          setDate={(date) =>
            setFilters({
              [startField]: date?.toISOString() as unknown as T[keyof T],
            } as Partial<T>)
          }
          label="Start Time"
        />
        <DateTimePicker
          date={endDate}
          setDate={(date) =>
            setFilters({
              [endField]: date?.toISOString() as unknown as T[keyof T],
            } as Partial<T>)
          }
          label="End Time"
        />
      </div>
    </div>
  );
}
