import { DateTimePicker } from '@/components/molecules/time-picker/date-time-picker';
import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import {
  TIME_PRESETS,
  useTimeFilters,
} from '@/next/hooks/utils/use-time-filters';
import { Pause } from 'lucide-react';
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from '@/next/components/ui/dropdown-menu';
import { ChevronDown } from 'lucide-react';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';

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
      className={cn('flex w-full items-center gap-2', className)}
      {...props}
    >
      {children}
    </div>
  );
}

export function TogglePause() {
  const { pause, resume, isPaused } = useTimeFilters();

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
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
            {isPaused ? 'Resume' : <Pause className="h-4 w-4" />}
          </Button>
        </TooltipTrigger>
        <TooltipContent>
          {isPaused ? 'Resume Live Data' : 'Pause Live Data'}
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function TimeFilter({ className }: TimeFilterProps) {
  const {
    filters,
    setTimeFilter,
    activePreset,
    handleTimeFilterChange,
    handleClearTimeFilters,
    pause,
    isPaused,
  } = useTimeFilters();

  const startDate = filters.startTime
    ? new Date(filters.startTime as string)
    : undefined;
  const endDate = filters.endTime
    ? new Date(filters.endTime as string)
    : undefined;

  return (
    <div className={cn('flex flex-col', className)}>
      {!isPaused ? (
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button
              variant="outline"
              size="sm"
              className="h-8 px-2 text-xs flex items-center gap-1"
            >
              Last {activePreset || 'Select Time Range'}
              <ChevronDown className="h-3 w-3" />
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {Object.entries(TIME_PRESETS).map(([key]) => (
              <DropdownMenuItem
                key={key}
                onClick={() =>
                  handleTimeFilterChange(key as keyof typeof TIME_PRESETS)
                }
                className={cn(
                  'cursor-pointer',
                  activePreset === key && 'bg-accent',
                )}
              >
                Last {key}
              </DropdownMenuItem>
            ))}
            <DropdownMenuItem onClick={() => pause()}>
              Custom Range
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      ) : (
        <div className="flex items-center gap-4">
          <DateTimePicker
            date={startDate}
            setDate={(date) => {
              if (date) {
                setTimeFilter({
                  startTime: date?.toISOString(),
                  endTime: endDate?.toISOString(),
                });
              } else {
                handleClearTimeFilters();
              }
            }}
            label="Start Time"
          />
          <DateTimePicker
            date={endDate}
            setDate={(date) => {
              if (date) {
                setTimeFilter({
                  startTime: startDate!.toISOString(),
                  endTime: date?.toISOString(),
                });
              } else {
                handleClearTimeFilters();
              }
            }}
            label="End Time"
          />
        </div>
      )}
    </div>
  );
}
