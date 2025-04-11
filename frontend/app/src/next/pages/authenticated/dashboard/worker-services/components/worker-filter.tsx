import { RadioGroup, RadioGroupItem } from '@/next/components/ui/radio-group';
import { Label } from '@/next/components/ui/label';
import { cn } from '@/next/lib/utils';

export type WorkerStatus = 'all' | 'active' | 'paused' | 'inactive';

interface WorkerFilterProps {
  selectedStatus: WorkerStatus;
  onStatusChange: (status: WorkerStatus) => void;
  counts: {
    all: number;
    active: number;
    paused: number;
    inactive: number;
  };
  className?: string;
}

export function WorkerFilter({
  selectedStatus,
  onStatusChange,
  counts,
  className,
}: WorkerFilterProps) {
  const filters: Array<{ value: WorkerStatus; label: string; count: number }> =
    [
      { value: 'all', label: 'All Workers', count: counts.all },
      { value: 'active', label: 'Active', count: counts.active },
      { value: 'paused', label: 'Paused', count: counts.paused },
      { value: 'inactive', label: 'Inactive', count: counts.inactive },
    ];

  return (
    <div className={cn('space-y-4', className)}>
      <RadioGroup
        value={selectedStatus}
        onValueChange={(value: string) => onStatusChange(value as WorkerStatus)}
        className="flex flex-wrap items-center gap-4"
      >
        {filters.map((filter) => (
          <div key={filter.value} className="flex items-center space-x-2">
            <RadioGroupItem
              value={filter.value}
              id={`filter-${filter.value}`}
            />
            <Label
              htmlFor={`filter-${filter.value}`}
              className="flex items-center"
            >
              {filter.label}
              <span className="ml-1.5 rounded-full bg-muted px-2 py-0.5 text-xs font-medium text-muted-foreground">
                {filter.count}
              </span>
            </Label>
          </div>
        ))}
      </RadioGroup>
    </div>
  );
}
