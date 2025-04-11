import * as React from 'react';
import { DateRange } from 'react-day-picker';
import { addDays, addHours, addMinutes, startOfDay } from 'date-fns';
import { ArrowLeftIcon } from 'lucide-react';

import { cn } from '@/next/lib/utils';
import { Button } from '@/next/components/ui/button';
import { DateRangePicker } from './date-range-picker';

export type RelativeTimeOption = {
  label: string;
  value: string;
  getDateRange: () => DateRange;
};

export interface TimeFilterProps {
  onChange: (range: DateRange | undefined) => void;
  value?: DateRange;
  className?: string;
  relativeOptions?: RelativeTimeOption[];
  hasCustomOption?: boolean;
  timezone?: string;
  customLabel?: string;
}

const defaultRelativeOptions: RelativeTimeOption[] = [
  {
    label: '1m',
    value: 'last-1-min',
    getDateRange: () => ({
      from: addMinutes(new Date(), -1),
      to: new Date(),
    }),
  },
  {
    label: '5m',
    value: 'last-5-min',
    getDateRange: () => ({
      from: addMinutes(new Date(), -5),
      to: new Date(),
    }),
  },
  {
    label: '30m',
    value: 'last-30-min',
    getDateRange: () => ({
      from: addMinutes(new Date(), -30),
      to: new Date(),
    }),
  },
  {
    label: '1h',
    value: 'last-hour',
    getDateRange: () => ({
      from: addHours(new Date(), -1),
      to: new Date(),
    }),
  },
  {
    label: '3h',
    value: 'last-3-hours',
    getDateRange: () => ({
      from: addHours(new Date(), -3),
      to: new Date(),
    }),
  },
  {
    label: '1d',
    value: 'last-day',
    getDateRange: () => ({
      from: addHours(new Date(), -24),
      to: new Date(),
    }),
  },
  {
    label: '7d',
    value: 'last-7-days',
    getDateRange: () => ({
      from: startOfDay(addDays(new Date(), -7)),
      to: new Date(),
    }),
  },
];

export function TimeFilter({
  onChange,
  value,
  className,
  relativeOptions = defaultRelativeOptions,
  hasCustomOption = true,
  timezone = Intl.DateTimeFormat().resolvedOptions().timeZone,
  customLabel = 'Custom',
}: TimeFilterProps) {
  const [isCustomSelected, setIsCustomSelected] = React.useState(false);
  const [selectedOption, setSelectedOption] = React.useState<string | null>(
    null,
  );

  // Apply a relative time option
  const handleRelativeOptionSelect = (option: RelativeTimeOption) => {
    const range = option.getDateRange();
    onChange(range);
    setSelectedOption(option.value);
    setIsCustomSelected(false);
  };

  // Switch to custom date range picker
  const handleCustomSelect = () => {
    setIsCustomSelected(true);
    setSelectedOption(null);
  };

  // Switch back to relative options
  const handleBackToRelative = () => {
    setIsCustomSelected(false);
  };

  // Find selected option based on value
  React.useEffect(() => {
    if (value && !isCustomSelected) {
      const match = findMatchingTimeOption(value, relativeOptions);
      if (match) {
        setSelectedOption(match.value);
      } else if (value && !selectedOption) {
        setSelectedOption('custom');
        setIsCustomSelected(true);
      }
    }
  }, [value, relativeOptions, isCustomSelected, selectedOption]);

  return (
    <div className={cn('space-y-3', className)}>
      {isCustomSelected ? (
        <div className="space-y-2">
          <div className="flex justify-between items-center">
            <Button
              variant="ghost"
              size="sm"
              onClick={handleBackToRelative}
              className="flex items-center gap-1 text-sm"
            >
              <ArrowLeftIcon className="h-3 w-3" />
              Back to presets
            </Button>
          </div>
          <DateRangePicker
            value={value}
            onChange={onChange}
            timezone={timezone}
          />
        </div>
      ) : (
        <div className="flex flex-wrap gap-2">
          {relativeOptions.map((option) => (
            <Button
              key={option.value}
              variant={selectedOption === option.value ? 'default' : 'outline'}
              size="sm"
              onClick={() => handleRelativeOptionSelect(option)}
            >
              {option.label}
            </Button>
          ))}
          {hasCustomOption && (
            <Button
              variant={isCustomSelected ? 'default' : 'outline'}
              size="sm"
              onClick={handleCustomSelect}
            >
              {customLabel}
            </Button>
          )}
        </div>
      )}
    </div>
  );
}

// Helper function to find a matching relative time option for a given date range
function findMatchingTimeOption(
  range: DateRange,
  options: RelativeTimeOption[],
): RelativeTimeOption | undefined {
  // Simple matching based on comparing timestamps
  // This is a rough approximation and may not always be accurate
  return options.find((option) => {
    const optionRange = option.getDateRange();
    const fromDiff = Math.abs(
      (optionRange.from?.getTime() || 0) - (range.from?.getTime() || 0),
    );
    const toDiff = Math.abs(
      (optionRange.to?.getTime() || 0) - (range.to?.getTime() || 0),
    );
    // Allow for a small deviation (1 second)
    return fromDiff < 1000 && toDiff < 1000;
  });
}
