import { TenantResource } from '@/lib/api';
import { cn } from '@/lib/utils';

export const limitedResources: Record<TenantResource, string> = {
  [TenantResource.WORKER]: 'Total Workers',
  [TenantResource.WORKER_SLOT]: 'Concurrency Slots',
  [TenantResource.EVENT]: 'Events',
  [TenantResource.TASK_RUN]: 'Task Runs',
  [TenantResource.CRON]: 'Cron Triggers',
  [TenantResource.SCHEDULE]: 'Schedule Triggers',
  [TenantResource.INCOMING_WEBHOOK]: 'Incoming Webhooks',
};

const indicatorVariants = {
  ok: 'border-transparent rounded-full bg-green-500',
  alarm: 'border-transparent rounded-full bg-yellow-500',
  exhausted: 'border-transparent rounded-full bg-red-500',
};

export function LimitIndicator({
  value,
  alarmValue,
  limitValue,
}: {
  value: number;
  alarmValue?: number;
  limitValue: number;
}) {
  let variant = indicatorVariants.ok;

  if (alarmValue && value >= alarmValue) {
    variant = indicatorVariants.alarm;
  }

  if (value >= limitValue) {
    variant = indicatorVariants.exhausted;
  }

  return <div className={cn(variant, 'h-[6px] w-[6px] rounded-full')} />;
}

export const limitDurationMap: Record<string, string> = {
  '24h0m0s': 'Daily',
  '168h0m0s': 'Weekly',
  '720h0m0s': 'Monthly',
};
