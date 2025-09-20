import { useMemo } from 'react';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { RefetchInterval, RefetchIntervalOption } from '@/lib/api/api';
import { RefreshCw } from 'lucide-react';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { Button } from './v1/ui/button';

type RefetchIntervalDropdownProps = {
  isRefetching: boolean;
  onRefetch: () => void;
};

export const RefetchIntervalDropdown = ({
  isRefetching,
  onRefetch,
}: RefetchIntervalDropdownProps) => {
  const { currentInterval, setRefetchInterval } = useRefetchInterval();

  const intervalOptions = useMemo(
    () =>
      Object.entries(RefetchInterval).map(([key, labeledInterval]) => ({
        key: key as RefetchIntervalOption,
        ...labeledInterval,
      })),
    [],
  );

  const handleValueChange = (selectedKey: string) => {
    const selectedOption = selectedKey as RefetchIntervalOption;
    const selectedInterval = RefetchInterval[selectedOption];
    setRefetchInterval(selectedInterval);
  };

  const value = useMemo(() => {
    return (
      Object.entries(RefetchInterval).find(
        ([, interval]) => interval.value === currentInterval.value,
      )?.[0] || 'off'
    );
  }, [currentInterval]);

  return (
    <div className="flex flex-row items-center h-8">
      <Button
        data-is-loading={isRefetching}
        className="h-full rounded-l-md rounded-r-none flex flex-row gap-x-2 data-[is-loading]:animate-spin"
        variant="outline"
        onClick={onRefetch}
      >
        <RefreshCw className="size-4" />
        Refresh
      </Button>
      <Select value={value} onValueChange={handleValueChange}>
        <SelectTrigger className="flex flex-row items-center gap-x-2 h-full rounded-r-md rounded-l-none border-l-0 hover:bg-accent">
          {value !== 'off' && (
            <div className="hidden cq-xl:inline">
              <SelectValue />
            </div>
          )}
        </SelectTrigger>
        <SelectContent>
          {intervalOptions.map(({ key, label }) => (
            <SelectItem key={key} value={key}>
              {label}
            </SelectItem>
          ))}
        </SelectContent>
      </Select>
    </div>
  );
};
