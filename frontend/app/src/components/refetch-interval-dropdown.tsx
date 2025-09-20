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

export const RefetchIntervalDropdown = () => {
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

  return (
    <Select
      value={
        Object.entries(RefetchInterval).find(
          ([, interval]) => interval.value === currentInterval.value,
        )?.[0] || 'off'
      }
      onValueChange={handleValueChange}
    >
      <SelectTrigger className="flex flex-row items-center gap-x-2 h-8">
        <RefreshCw className="size-4" />
        <div className="hidden cq-xl:inline">
          <SelectValue />
        </div>
      </SelectTrigger>
      <SelectContent>
        {intervalOptions.map(({ key, label }) => (
          <SelectItem key={key} value={key}>
            {label}
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
};
