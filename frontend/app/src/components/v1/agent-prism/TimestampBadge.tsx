import type { BadgeProps } from './Badge';
import { Badge } from './Badge';
import type { ComponentPropsWithRef } from 'react';

export type TimestampBadgeProps = ComponentPropsWithRef<'span'> & {
  timestamp: number;
  size?: BadgeProps['size'];
};

export const TimestampBadge = ({
  timestamp,
  size,
  ...rest
}: TimestampBadgeProps) => {
  return <Badge size={size} {...rest} label={formatTimestamp(timestamp)} />;
};

function formatTimestamp(timestamp: number): string {
  return new Date(timestamp).toLocaleString();
}
