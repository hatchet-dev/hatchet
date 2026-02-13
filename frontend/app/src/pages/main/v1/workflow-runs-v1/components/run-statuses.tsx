import { V1TaskStatus } from '@/lib/api';
import { cn } from '@/lib/utils';
import { CircleMinus } from 'lucide-react';

function createV2IndicatorVariant(eventType: V1TaskStatus | undefined) {
  switch (eventType) {
    case V1TaskStatus.CANCELLED:
      return 'border-transparent rounded-full bg-orange-500';
    case V1TaskStatus.FAILED:
      return 'border-transparent rounded-full bg-red-500';
    case V1TaskStatus.RUNNING:
      return 'border-transparent rounded-full bg-yellow-500';
    case V1TaskStatus.QUEUED:
      return 'border-transparent rounded-full bg-slate-500';
    case V1TaskStatus.COMPLETED:
      return 'border-transparent rounded-full bg-green-500';
    default:
      return 'border-transparent rounded-full bg-muted';
  }
}

export function V1RunIndicator({
  status,
  isSkipped,
}: {
  status: V1TaskStatus | undefined;
  isSkipped?: boolean;
}) {
  if (isSkipped) {
    return <CircleMinus className="h-3 w-3 text-muted-foreground" />;
  }

  const indicator = createV2IndicatorVariant(status);

  return <div className={cn(indicator, 'h-[6px] w-[6px] rounded-full')} />;
}
