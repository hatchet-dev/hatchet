import { TenantResource } from '@/lib/api';
import { getResourceLimitStatus } from '@/lib/resource-limit-status';
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
  warn: 'border-transparent rounded-full bg-yellow-500',
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
  const status = getResourceLimitStatus({ value, alarmValue, limitValue });

  return (
    <div
      className={cn(indicatorVariants[status], 'h-[6px] w-[6px] rounded-full')}
    />
  );
}

export const limitDurationMap: Record<string, string> = {
  '24h0m0s': 'Daily',
  '168h0m0s': 'Weekly',
  '720h0m0s': 'Monthly',
};
