import { Button } from './v1/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import {
  RefetchInterval,
  RefetchIntervalOption,
} from '@/lib/api/refetch-interval';
import { RefreshCw } from 'lucide-react';
import { useMemo } from 'react';

type RefetchIntervalDropdownProps = {
  isRefetching: boolean;
  onRefetch: () => void;
};

export const RefetchIntervalDropdown = ({
  isRefetching,
  onRefetch,
}: RefetchIntervalDropdownProps) => {
  const { userRefetchIntervalPreference, setRefetchInterval } =
    useRefetchInterval();

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
        ([, interval]) =>
          interval.value === userRefetchIntervalPreference.value,
      )?.[0] || 'off'
    );
  }, [userRefetchIntervalPreference]);

  return (
    <div className="flex h-8 flex-row items-center">
      <Button
        className="flex h-full flex-row gap-x-2 rounded-l-md rounded-r-none pl-3"
        variant="outline"
        onClick={onRefetch}
      >
        <RefreshCw
          data-is-refetching={isRefetching}
          className="size-4 data-[is-refetching=true]:animate-spin"
        />
        <span className="cq-xl:inline hidden">Refresh</span>
      </Button>
      <Select value={value} onValueChange={handleValueChange}>
        <SelectTrigger className="flex h-full flex-row items-center gap-x-2 rounded-l-none rounded-r-md border-l-0 hover:bg-accent">
          {value !== 'off' && <SelectValue />}
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
