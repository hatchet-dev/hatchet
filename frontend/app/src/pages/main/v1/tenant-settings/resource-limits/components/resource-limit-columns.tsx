import RelativeDate from '@/components/v1/molecules/relative-date';
import { TenantResource, TenantResourceLimit } from '@/lib/api';
import { getResourceLimitStatus } from '@/lib/resource-limit-status';
import { cn } from '@/lib/utils';
import { useMemo } from 'react';

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

export function useResourceLimitColumns() {
  return useMemo(
    () => [
      {
        columnLabel: 'Resource',
        cellRenderer: (limit: TenantResourceLimit) => (
          <div className="flex flex-row items-center gap-3">
            <LimitIndicator
              value={limit.value}
              alarmValue={limit.alarmValue}
              limitValue={limit.limitValue}
            />
            <span className="font-medium text-foreground">
              {limitedResources[limit.resource]}
            </span>
          </div>
        ),
      },
      {
        columnLabel: 'Current Value',
        cellRenderer: (limit: TenantResourceLimit) => (
          <span className="tabular-nums">{limit.value}</span>
        ),
      },
      {
        columnLabel: 'Limit Value',
        cellRenderer: (limit: TenantResourceLimit) => (
          <span className="tabular-nums">{limit.limitValue}</span>
        ),
      },
      {
        columnLabel: 'Alarm Value',
        cellRenderer: (limit: TenantResourceLimit) => (
          <span className="tabular-nums">{limit.alarmValue || 'N/A'}</span>
        ),
      },
      {
        columnLabel: 'Meter Window',
        cellRenderer: (limit: TenantResourceLimit) =>
          (limit.window || '-') in limitDurationMap
            ? limitDurationMap[limit.window || '-']
            : limit.window,
      },
      {
        columnLabel: 'Last Refill',
        cellRenderer: (limit: TenantResourceLimit) =>
          !limit.window
            ? 'N/A'
            : limit.lastRefill && <RelativeDate date={limit.lastRefill} />,
      },
    ],
    [],
  );
}
