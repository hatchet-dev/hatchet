import { LogsChart } from './components/logs-chart';
import { useTenantLogs } from './hooks/use-tenant-logs';
import type { TimeWindow } from './hooks/use-tenant-logs';
import { RefetchIntervalDropdown } from '@/components/refetch-interval-dropdown';
import {
  getAutocomplete,
  applySuggestion,
} from '@/components/v1/cloud/logging/log-search/autocomplete';
import type { LogAutocompleteContext } from '@/components/v1/cloud/logging/log-search/autocomplete';
import type { AutocompleteSuggestion } from '@/components/v1/cloud/logging/log-search/types';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { RetentionBanner } from '@/components/v1/retention-banner';
import { Button } from '@/components/v1/ui/button';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/v1/ui/select';
import { Skeleton } from '@/components/v1/ui/skeleton';
import { FeatureFlagId, useIsFeatureEnabled } from '@/hooks/use-feature-flags';
import { useSidePanel } from '@/hooks/use-side-panel';
import { isBeforeRetention } from '@/lib/utils/retention';
import { useAppContext } from '@/providers/app-context';
import { XCircleIcon } from 'lucide-react';
import { useCallback, useMemo } from 'react';

export default function TenantLogsPage() {
  const { tenant } = useAppContext();
  const { isEnabled: isWorkflowFilterEnabled } = useIsFeatureEnabled(
    FeatureFlagId.TenantLogWorkflowFilterEnabled,
    true,
  );

  const {
    logs,
    isLoading,
    isRefetching,
    refetch,
    fetchOlderLogs,
    queryString,
    setQueryString,
    chartSince,
    chartUntil,
    chartMetrics,
    setCustomTimeRange,
    timeWindow,
    isCustomTimeRange,
    customSince,
    customUntil,
    setTimeWindow,
    clearTimeRange,
    setCustomSince,
    setCustomUntil,
    workflowNames,
  } = useTenantLogs();

  const sidePanel = useSidePanel();

  const autocompleteContext = useMemo<LogAutocompleteContext>(
    () => ({ workflowNames: isWorkflowFilterEnabled ? workflowNames : [] }),
    [workflowNames, isWorkflowFilterEnabled],
  );

  const handleViewRun = useCallback(
    (taskRunId: string) => {
      sidePanel.open({
        type: 'task-run-details',
        content: {
          taskRunId,
          showViewTaskRunButton: true,
        },
      });
    },
    [sidePanel],
  );

  return (
    <div className="flex flex-col h-full gap-4">
      <LogsChart
        metrics={chartMetrics}
        since={chartSince}
        until={chartUntil}
        onZoom={setCustomTimeRange}
      />
      <div className="flex items-center gap-2 shrink-0">
        <SearchBarWithFilters<AutocompleteSuggestion, LogAutocompleteContext>
          value={queryString}
          onChange={setQueryString}
          onSubmit={setQueryString}
          getAutocomplete={(q, ctx) => {
            const result = getAutocomplete(q, ctx);
            return {
              ...result,
              suggestions: result.suggestions.filter(
                (s) =>
                  s.value !== 'attempt:' &&
                  (isWorkflowFilterEnabled || s.value !== 'workflow:'), // only show workflow filter if feature is enabled
              ),
            };
          }}
          applySuggestion={applySuggestion}
          autocompleteContext={autocompleteContext}
          placeholder="Search logs..."
          filterChips={[
            {
              key: 'level:',
              label: 'Level',
              description: 'Filter by log level',
            },
            ...(isWorkflowFilterEnabled
              ? [
                  {
                    key: 'workflow:',
                    label: 'Workflow',
                    description: 'Filter by workflow name',
                  },
                ]
              : []),
          ]}
          className="flex-1"
        />
        {isCustomTimeRange ? (
          <div className="flex items-center gap-2">
            <DateTimePicker
              label="After"
              date={customSince ? new Date(customSince) : undefined}
              setDate={(date) => setCustomSince(date?.toISOString())}
              triggerClassName="h-8 text-xs"
            />
            <DateTimePicker
              label="Before"
              date={customUntil ? new Date(customUntil) : undefined}
              setDate={(date) => setCustomUntil(date?.toISOString())}
              triggerClassName="h-8 text-xs"
            />
            <Button
              onClick={clearTimeRange}
              variant="ghost"
              size="sm"
              leftIcon={<XCircleIcon className="size-4" />}
            >
              Clear
            </Button>
          </div>
        ) : (
          <Select
            value={timeWindow}
            onValueChange={(value) => {
              if (value === 'custom') {
                setCustomTimeRange(chartSince, new Date().toISOString());
              } else {
                setTimeWindow(value as TimeWindow);
              }
            }}
          >
            <SelectTrigger className="h-8 text-xs w-28">
              <SelectValue placeholder="Choose time range" />
            </SelectTrigger>
            <SelectContent>
              <SelectItem value="1h">1 hour</SelectItem>
              <SelectItem value="6h">6 hours</SelectItem>
              <SelectItem value="1d">1 day</SelectItem>
              <SelectItem value="7d">7 days</SelectItem>
              <SelectItem value="custom">Custom</SelectItem>
            </SelectContent>
          </Select>
        )}
        <RefetchIntervalDropdown
          isRefetching={isRefetching}
          onRefetch={refetch}
        />
      </div>
      {tenant?.dataRetentionPeriod &&
      isBeforeRetention(chartSince, tenant.dataRetentionPeriod) ? (
        <div className="relative flex-1">
          <div className="pointer-events-none select-none blur-[3px] opacity-40">
            {Array.from({ length: 12 }).map((_, i) => (
              <div
                key={i}
                className="flex items-center gap-3 border-b border-border/40 px-4 py-2"
              >
                <Skeleton className="h-3 w-16" />
                <Skeleton className="h-3 w-12" />
                <Skeleton className="h-3 flex-1" />
              </div>
            ))}
          </div>
          <div className="absolute inset-0 flex items-center justify-center p-8">
            <div className="w-full max-w-lg">
              <RetentionBanner retentionPeriod={tenant.dataRetentionPeriod} />
            </div>
          </div>
        </div>
      ) : (
        <LogViewer
          key={queryString + chartSince + (chartUntil ?? '')}
          logs={logs}
          onScrollToBottom={fetchOlderLogs}
          isLoading={isLoading}
          onViewRun={handleViewRun}
          showAttempt={false}
          showTaskName
          emptyMessage="No logs found for this time window."
        />
      )}
    </div>
  );
}
