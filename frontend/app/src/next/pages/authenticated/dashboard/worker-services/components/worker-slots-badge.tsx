import { Badge } from '@/next/components/ui/badge';
import { cn } from '@/next/lib/utils';

interface SlotsBadgeProps {
  available: number;
  max: number;
}

export function slotsColor(available: number, max: number) {
  // Calculate percentage
  const percentage = max > 0 ? (available / max) * 100 : 0;

  // Determine color based on percentage
  if (percentage > 50) {
    return 'bg-green-50 text-green-700 border-green-200';
  } else if (percentage >= 30) {
    return 'bg-yellow-50 text-yellow-700 border-yellow-200';
  } else {
    return 'bg-red-50 text-red-700 border-red-200';
  }
}

export function SlotsBadge({ available, max }: SlotsBadgeProps) {
  return (
    <Badge variant="outline" className={cn(slotsColor(available, max))}>
      {available} / {max}
    </Badge>
  );
}
