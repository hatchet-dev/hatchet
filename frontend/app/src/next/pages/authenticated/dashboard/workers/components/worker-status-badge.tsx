import { Badge, BadgeProps } from '@/next/components/ui/badge';
import { cn } from '@/next/lib/utils';

interface WorkerStatusBadgeProps extends BadgeProps {
  status?: string;
  count?: number;
  animated?: boolean;
  isLoading?: boolean;
}

type StatusConfig = {
  colors: string;
  primary: string;
  primaryOKLCH: string;
  label: string;
};

export const WorkerStatusConfigs: Record<string, StatusConfig> = {
  ACTIVE: {
    colors:
      'text-green-800 dark:text-green-300 bg-green-500/20 ring-green-500/30',
    primary: 'text-green-500 bg-green-500',
    primaryOKLCH: 'oklch(0.723 0.219 149.579)',
    label: 'Active',
  },
  INACTIVE: {
    colors: 'text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500',
    primary: 'text-red-500 bg-red-500',
    primaryOKLCH: 'oklch(0.637 0.237 25.331)',
    label: 'Inactive',
  },
  PAUSED: {
    colors:
      'text-yellow-800 dark:text-yellow-300 bg-yellow-500/20 ring-yellow-500/30',
    primary: 'text-yellow-500 bg-yellow-500',
    primaryOKLCH: 'oklch(0.795 0.184 86.047)',
    label: 'Paused',
  },
};

export function WorkerStatusBadge({
  status,
  count,
  variant,
  animated,
  isLoading,
  className,
  ...props
}: WorkerStatusBadgeProps) {
  const config = !status
    ? {
        colors:
          'text-gray-800 dark:text-gray-300 bg-gray-500/20 ring-gray-500/30',
        primary: 'text-gray-500 bg-gray-500',
        primaryOKLCH: 'oklch(0.551 0.027 264.364)',
        label: 'Unknown',
      }
    : WorkerStatusConfigs[status] || {
        colors:
          'text-gray-800 dark:text-gray-300 bg-gray-500/20 ring-gray-500/30',
        primary: 'text-gray-500 bg-gray-500',
        primaryOKLCH: 'oklch(0.551 0.027 264.364)',
        label: status,
      };

  const isDisabled = count === 0;
  const finalConfig = isDisabled
    ? {
        colors: 'text-red-800 dark:text-red-300 bg-red-500/20 ring-red-500',
        primary: 'text-red-500 bg-red-500',
      }
    : config;

  const content =
    variant !== 'xs' ? (
      <>
        {count !== undefined && `${count} `}
        {config.label}
      </>
    ) : null;

  return (
    <Badge
      className={cn(
        variant === 'xs' ? 'p-0 w-2 h-2' : 'px-3 py-1',
        variant === 'xs' ? finalConfig.primary : finalConfig.colors,
        'text-xs font-medium rounded-md border-transparent',
        className,
      )}
      tooltipContent={status}
      animated={false}
      variant={variant}
      {...props}
    >
      {content}
    </Badge>
  );
}
