import {
  getAutocomplete,
  applySuggestion,
} from '@/components/v1/cloud/logging/log-search/autocomplete';
import { parseLogQuery } from '@/components/v1/cloud/logging/log-search/parser';
import type { AutocompleteSuggestion } from '@/components/v1/cloud/logging/log-search/types';
import { LOG_LEVEL_TO_API } from '@/components/v1/cloud/logging/log-search/types';
import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useSidePanel } from '@/hooks/use-side-panel';
import { V1LogLine, V1LogLineOrderByDirection } from '@/lib/api';
import api from '@/lib/api/api';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useCallback, useMemo, useState } from 'react';
import { TabOption } from './step-run-detail/step-run-detail';

const LOGS_PER_PAGE = 100;

function logKey(log: LogLine): string {
  return `${log.timestamp ?? ''}-${log.line ?? ''}`;
}

function mapToLogLines(rows: V1LogLine[]): LogLine[] {
  return rows.map((row) => ({
    timestamp: row.createdAt,
    line: row.message,
    level: row.level,
    attempt: row.attempt,
    taskExternalId: row.taskExternalId,
    taskDisplayName: row.taskDisplayName,
  }));
}

interface WorkflowRunLogsProps {
  taskExternalIds: string[];
}

export function WorkflowRunLogs({ taskExternalIds }: WorkflowRunLogsProps) {
  const { tenantId } = useCurrentTenantId();
  const sidePanel = useSidePanel();
  const [queryString, setQueryString] = useState('');
  const parsedQuery = useMemo(() => parseLogQuery(queryString), [queryString]);

  // Stable string key so the query key doesn't use an array reference
  const taskExternalIdsKey = taskExternalIds.join(',');

  const logsQuery = useInfiniteQuery({
    queryKey: [
      'workflow-run-logs',
      taskExternalIdsKey,
      parsedQuery.level,
      parsedQuery.search,
      parsedQuery.attempt,
    ],
    queryFn: async ({ pageParam }: { pageParam: string | undefined }) => {
      const response = await api.v1TenantLogLineList(tenantId, {
        limit: LOGS_PER_PAGE,
        taskExternalIds,
        ...(pageParam && { until: pageParam }),
        ...(parsedQuery.level && {
          levels: [LOG_LEVEL_TO_API[parsedQuery.level]],
        }),
        ...(parsedQuery.search && { search: parsedQuery.search }),
        ...(parsedQuery.attempt && { attempt: parsedQuery.attempt }),
        order_by_direction: V1LogLineOrderByDirection.DESC,
      });
      return response.data;
    },
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (lastPage) => {
      const rows = lastPage.rows ?? [];
      if (rows.length < LOGS_PER_PAGE) return undefined;
      return rows[rows.length - 1].createdAt;
    },
    enabled: !!tenantId && taskExternalIds.length > 0,
  });

  const logs = useMemo<LogLine[]>(() => {
    const seen = new Set<string>();
    const result: LogLine[] = [];
    for (const page of logsQuery.data?.pages ?? []) {
      for (const log of mapToLogLines(page.rows ?? [])) {
        const k = logKey(log);
        if (!seen.has(k)) {
          seen.add(k);
          result.push(log);
        }
      }
    }
    return result;
  }, [logsQuery.data]);

  const fetchOlderLogs = useCallback(() => {
    if (logsQuery.hasNextPage && !logsQuery.isFetchingNextPage) {
      logsQuery.fetchNextPage();
    }
  }, [logsQuery]);

  const handleViewRun = useCallback(
    (taskRunId: string) => {
      sidePanel.open({
        type: 'task-run-details',
        content: {
          taskRunId,
          defaultOpenTab: TabOption.Output,
          showViewTaskRunButton: true,
        },
      });
    },
    [sidePanel],
  );

  return (
    <div className="my-4 flex flex-col gap-y-2 max-h-[40rem] min-h-[25rem]">
      <SearchBarWithFilters<AutocompleteSuggestion, number[]>
        value={queryString}
        onChange={setQueryString}
        onSubmit={setQueryString}
        getAutocomplete={(q) => getAutocomplete(q, [])}
        applySuggestion={applySuggestion}
        autocompleteContext={[]}
        placeholder="Search logs..."
        filterChips={[
          { key: 'level:', label: 'Level', description: 'Filter by log level' },
          { key: 'attempt:', label: 'Attempt', description: 'Filter by attempt number' },
        ]}
      />
      <LogViewer
        key={queryString + taskExternalIdsKey}
        logs={logs}
        onScrollToBottom={fetchOlderLogs}
        isLoading={logsQuery.isLoading}
        onViewRun={handleViewRun}
        showTaskName
        emptyMessage="No logs found for this run."
      />
    </div>
  );
}
