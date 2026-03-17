import {
  getAutocomplete,
  applySuggestion,
} from '@/components/v1/cloud/logging/log-search/autocomplete';
import { parseLogQuery } from '@/components/v1/cloud/logging/log-search/parser';
import type { AutocompleteSuggestion } from '@/components/v1/cloud/logging/log-search/types';
import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { useSidePanel } from '@/hooks/use-side-panel';
import { V1LogLine, V1LogLineLevel, V1LogLineOrderByDirection } from '@/lib/api';
import api from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
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

  const [mergedLogs, setMergedLogs] = useState<LogLine[]>([]);
  const [isFetchingMore, setIsFetchingMore] = useState(false);
  const [hasMore, setHasMore] = useState(true);
  const lastDataUpdatedAt = useRef<number | undefined>(undefined);

  const taskExternalIdsKey = taskExternalIds.join(',');

  // Reset when query or task list changes
  useEffect(() => {
    setMergedLogs([]);
    setHasMore(true);
    lastDataUpdatedAt.current = undefined;
  }, [queryString, taskExternalIdsKey, parsedQuery.attempt]);

  const logsQuery = useQuery({
    queryKey: [
      'workflow-run-logs',
      taskExternalIds,
      parsedQuery.level,
      parsedQuery.search,
      parsedQuery.attempt,
    ],
    queryFn: async () => {
      const response = await api.v1TenantLogLineList(tenantId, {
        limit: LOGS_PER_PAGE,
        taskExternalIds,
        ...(parsedQuery.level && {
          levels: [parsedQuery.level.toUpperCase() as V1LogLineLevel],
        }),
        ...(parsedQuery.search && { search: parsedQuery.search }),
        ...(parsedQuery.attempt && { attempt: parsedQuery.attempt }),
        order_by_direction: V1LogLineOrderByDirection.DESC,
      });
      return response.data;
    },
    enabled: !!tenantId && taskExternalIds.length > 0,
  });

  useEffect(() => {
    if (!logsQuery.isSuccess) return;
    if (logsQuery.dataUpdatedAt === lastDataUpdatedAt.current) return;
    lastDataUpdatedAt.current = logsQuery.dataUpdatedAt;

    const incoming = mapToLogLines(logsQuery.data?.rows ?? []);
    setMergedLogs((prev) => {
      if (prev.length === 0) return incoming;
      const existingKeys = new Set(prev.map(logKey));
      const fresh = incoming.filter((l) => !existingKeys.has(logKey(l)));
      return [...fresh, ...prev];
    });
  }, [logsQuery.isSuccess, logsQuery.dataUpdatedAt, logsQuery.data]);

  const fetchOlderLogs = useCallback(async () => {
    if (isFetchingMore || !hasMore) return;
    const oldest = mergedLogs[mergedLogs.length - 1];
    if (!oldest?.timestamp) return;

    setIsFetchingMore(true);
    try {
      const response = await api.v1TenantLogLineList(tenantId, {
        limit: LOGS_PER_PAGE,
        taskExternalIds,
        until: oldest.timestamp,
        ...(parsedQuery.level && {
          levels: [parsedQuery.level.toUpperCase() as V1LogLineLevel],
        }),
        ...(parsedQuery.search && { search: parsedQuery.search }),
        ...(parsedQuery.attempt && { attempt: parsedQuery.attempt }),
        order_by_direction: V1LogLineOrderByDirection.DESC,
      });
      const older = mapToLogLines(response.data.rows ?? []);
      if (older.length < LOGS_PER_PAGE) setHasMore(false);
      if (older.length > 0) {
        setMergedLogs((prev) => {
          const existingKeys = new Set(prev.map(logKey));
          return [...prev, ...older.filter((l) => !existingKeys.has(logKey(l)))];
        });
      }
    } finally {
      setIsFetchingMore(false);
    }
  }, [tenantId, taskExternalIds, isFetchingMore, hasMore, mergedLogs, parsedQuery]);

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
        key={queryString}
        logs={mergedLogs}
        onScrollToBottom={fetchOlderLogs}
        isLoading={logsQuery.isLoading && mergedLogs.length === 0}
        onViewRun={handleViewRun}
        emptyMessage="No logs found for this run."
      />
    </div>
  );
}
