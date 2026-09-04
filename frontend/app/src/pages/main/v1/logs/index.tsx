import { RunsEmptyGraphic } from '../workflow-runs-v1/components/runs-empty-graphic';
import { RequestTimeoutCloudCTAEmptyState } from '../workflow-runs-v1/components/runs-timeout-empty-state';
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
import { EmptyState } from '@/components/v1/molecules/empty-state/empty-state';
import { WorkflowsGuard } from '@/components/v1/molecules/empty-state/workflows-guard';
import { RetentionUpgradeDialog } from '@/components/v1/retention-upgrade-dialog';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { DateTimePicker } from '@/components/v1/molecules/time-picker/date-time-picker';
import { TimeRangeSelect } from '@/components/v1/molecules/time-picker/time-range-select';
import { Button } from '@/components/v1/ui/button';
import { useSidePanel } from '@/hooks/use-side-panel';
import { docsPages } from '@/lib/generated/docs';
import { formatRetentionPeriod } from '@/lib/utils/retention';
import { XCircleIcon } from 'lucide-react';
import { useCallback, useMemo } from 'react';

export default function TenantLogsPage() {
  return (
    <WorkflowsGuard
      title="No logs found"
      description="Logs are emitted by your workers as they execute tasks. Run a task to see its logs appear here."
      docs={{
        href: docsPages.v1.logging.href,
        description: 'Learn about logging',
      }}
    >
      <TenantLogs />
    </WorkflowsGuard>
  );
}

function TenantLogs() {
  const {
    logs,
    fetchTimedOut,
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
    hasActiveFilters,
    isDefaultOneDayWindow,
    resetFilters,
    searchAllRetainedHistory,
    retentionPeriod,
    retentionGate,
  } = useTenantLogs();
  const retainedLabel = retentionPeriod
    ? formatRetentionPeriod(retentionPeriod)
    : '1 day';

  const sidePanel = useSidePanel();

  const autocompleteContext = useMemo<LogAutocompleteContext>(
    () => ({ workflowNames }),
    [workflowNames],
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
      <RetentionUpgradeDialog
        attempt={retentionGate.attempt}
        retentionPeriod={retentionPeriod}
        onClose={retentionGate.close}
      />
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
                (s) => s.value !== 'attempt:',
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
            {
              key: 'workflow:',
              label: 'Workflow',
              description: 'Filter by workflow name',
            },
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
              retentionPeriod={retentionPeriod}
              onBlockedDate={retentionGate.blockSince}
            />
            <DateTimePicker
              label="Before"
              date={customUntil ? new Date(customUntil) : undefined}
              setDate={(date) => setCustomUntil(date?.toISOString())}
              triggerClassName="h-8 text-xs"
              retentionPeriod={retentionPeriod}
              onBlockedDate={retentionGate.blockSince}
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
          <TimeRangeSelect
            value={timeWindow}
            onChange={(value) => {
              if (value === 'custom') {
                setCustomTimeRange(chartSince, new Date().toISOString());
              } else {
                setTimeWindow(value as TimeWindow);
              }
            }}
            retentionPeriod={retentionPeriod}
            triggerClassName="h-8 w-28 text-xs"
          />
        )}
        <RefetchIntervalDropdown
          isRefetching={isRefetching}
          onRefetch={refetch}
        />
      </div>
      <LogViewer
        key={queryString + chartSince + (chartUntil ?? '')}
        logs={logs}
        onScrollToBottom={fetchOlderLogs}
        isLoading={isLoading}
        onViewRun={handleViewRun}
        showAttempt={false}
        showTaskName
        emptyComponent={
          fetchTimedOut ? (
            <RequestTimeoutCloudCTAEmptyState utmCampaign="logs" />
          ) : hasActiveFilters ? (
            <EmptyState
              graphic={<RunsEmptyGraphic />}
              title="No logs matching your filters"
              buttons={[{ label: 'Clear filters', onClick: resetFilters }]}
            />
          ) : (
            <EmptyState
              title="No logs found"
              description={`No logs in the last ${retainedLabel}.`}
              links={[
                {
                  href: docsPages.v1.logging.href,
                  label: 'Learn about logging',
                  external: true,
                },
              ]}
              buttons={
                isDefaultOneDayWindow
                  ? [
                      {
                        label: 'Search all retained history',
                        onClick: searchAllRetainedHistory,
                      },
                    ]
                  : undefined
              }
            />
          )
        }
      />
    </div>
  );
}
