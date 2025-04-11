import { Badge, BadgeProps } from '@/next/components/ui/badge';

interface WorkerStatusBadgeProps extends BadgeProps {
  status?: string;
  count?: number;
}

type StatusConfig = {
  colors: string;
  label: string;
};

export const WorkerStatusConfigs: Record<string, StatusConfig> = {
  ACTIVE: {
    colors: 'bg-green-500 text-white border-green-600',
    label: 'Active',
  },
  INACTIVE: {
    colors: 'bg-red-500 text-white border-red-600',
    label: 'Inactive',
  },
  PAUSED: {
    colors: 'bg-yellow-500 text-white border-yellow-600',
    label: 'Paused',
  },
};

export function WorkerStatusBadge({
  status,
  count,
  variant,
  ...props
}: WorkerStatusBadgeProps) {
  const config = !status
    ? { colors: 'bg-gray-50 text-gray-700 border-gray-200', label: 'Unknown' }
    : WorkerStatusConfigs[status] || {
        colors: 'bg-gray-50 text-gray-700 border-gray-200',
        label: status,
      };

  const isDisabled = count === 0;
  const disabledClass = isDisabled
    ? 'bg-red-50 text-red-700 border-red-200'
    : config.colors;

  return (
    <Badge
      className={disabledClass}
      tooltipContent={status}
      variant={variant}
      {...props}
    >
      {variant !== 'xs' && (
        <>
          {count !== undefined && `${count} `}
          {config.label}
        </>
      )}
    </Badge>
  );
}
