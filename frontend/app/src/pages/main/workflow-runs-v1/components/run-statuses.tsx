import { V1TaskStatus } from '@/lib/api';
import { cn } from '@/lib/utils';

function createV2IndicatorVariant(eventType: V1TaskStatus | undefined) {
  switch (eventType) {
    case V1TaskStatus.CANCELLED:
      return 'border-transparent rounded-full bg-orange-500';
    case V1TaskStatus.FAILED:
      return 'border-transparent rounded-full bg-red-500';
    case V1TaskStatus.RUNNING:
      return 'border-transparent rounded-full bg-yellow-500';
    case V1TaskStatus.QUEUED:
      return 'border-transparent rounded-full bg-fuchsia-500';
    case V1TaskStatus.COMPLETED:
      return 'border-transparent rounded-full bg-green-500';
    default:
      return 'border-transparent rounded-full bg-muted';
  }
}

export function V1RunIndicator({
  status,
}: {
  status: V1TaskStatus | undefined;
}) {
  const indicator = createV2IndicatorVariant(status);

  return <div className={cn(indicator, 'rounded-full h-[6px] w-[6px]')} />;
}
