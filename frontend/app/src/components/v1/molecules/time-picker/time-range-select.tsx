import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import useControlPlane from '@/hooks/use-control-plane';
import {
  TIME_WINDOW_LABELS,
  TIME_WINDOW_PRESETS,
  isTimeWindowOutsideRetention,
  type TimeWindowPreset,
} from '@/lib/utils/retention';
import { Lock } from 'lucide-react';

const PRESETS: { value: TimeWindowPreset | 'custom'; label: string }[] = [
  ...TIME_WINDOW_PRESETS.map((value) => ({
    value,
    label: TIME_WINDOW_LABELS[value],
  })),
  { value: 'custom', label: 'Custom' },
];

type TimeRangeSelectProps = {
  value: string;
  onChange: (value: string) => void;
  retentionPeriod?: string;
  triggerClassName?: string;
};

export function TimeRangeSelect({
  value,
  onChange,
  retentionPeriod,
  triggerClassName,
}: TimeRangeSelectProps) {
  const { isControlPlaneEnabled } = useControlPlane();

  return (
    <div className="space-y-1">
      <Select value={value} onValueChange={onChange}>
        <SelectTrigger className={triggerClassName ?? 'h-8 text-xs'}>
          <SelectValue placeholder="Choose time range" />
        </SelectTrigger>
        <SelectContent>
          {PRESETS.map((preset) => {
            const locked =
              !!retentionPeriod &&
              preset.value !== 'custom' &&
              isTimeWindowOutsideRetention(preset.value, retentionPeriod);

            return (
              <SelectItem key={preset.value} value={preset.value}>
                <span className="flex w-full items-center justify-between gap-6">
                  <span>{preset.label}</span>
                  {locked ? (
                    <span className="flex items-center gap-1 text-muted-foreground">
                      <Lock className="size-3" />
                      {isControlPlaneEnabled ? (
                        <span className="text-[10px] uppercase tracking-wide">
                          Upgrade
                        </span>
                      ) : null}
                    </span>
                  ) : null}
                </span>
              </SelectItem>
            );
          })}
        </SelectContent>
      </Select>
    </div>
  );
}
