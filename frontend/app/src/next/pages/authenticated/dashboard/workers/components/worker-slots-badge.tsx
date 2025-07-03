import { Badge, BadgeProps } from '@/next/components/ui/badge';
import { cn } from '@/next/lib/utils';

interface SlotsBadgeProps extends BadgeProps {
  available: number;
  max: number;
  animated?: boolean;
  isLoading?: boolean;
}

type SlotsConfig = {
  colors: string;
  primary: string;
  primaryOKLCH: string;
};

export function getSlotsConfig(available: number, max: number): SlotsConfig {
  // Calculate percentage
  const percentage = max > 0 ? (available / max) * 100 : 0;

  // Determine config based on percentage
  if (percentage > 50) {
    return {
      colors:
        'text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
      primary: 'text-green-500 bg-green-500',
      primaryOKLCH: 'oklch(0.723 0.219 149.579)',
    };
  } else if (percentage >= 30) {
    return {
      colors:
        'text-yellow-800 dark:text-yellow-300 bg-yellow-500/20 ring-yellow-500/30',
      primary: 'text-yellow-500 bg-yellow-500',
      primaryOKLCH: 'oklch(0.795 0.184 86.047)',
    };
  } else {
    return {
      colors: 'text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500',
      primary: 'text-red-500 bg-red-500',
      primaryOKLCH: 'oklch(0.637 0.237 25.331)',
    };
  }
}

export function SlotsBadge({
  available,
  max,
  variant,
  animated,
  isLoading,
  className,
  ...props
}: SlotsBadgeProps) {
  const config = getSlotsConfig(available, max);

  const content = variant !== 'xs' ? `${available} / ${max}` : null;

  return (
    <Badge
      className={cn(
        variant === 'xs' ? 'p-0 w-2 h-2' : 'px-3 py-1',
        variant === 'xs' ? config.primary : config.colors,
        'text-xs font-medium rounded-md border-transparent',
        className,
      )}
      tooltipContent={`${available} of ${max} slots available`}
      animated={isLoading ? false : animated !== undefined ? animated : false}
      variant={variant}
      {...props}
    >
      {content}
    </Badge>
  );
}
