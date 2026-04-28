import { TabOption } from './step-run-detail/step-run-detail';
import {
  getAutocomplete,
  applySuggestion,
} from '@/components/v1/cloud/logging/log-search/autocomplete';
import type { LogAutocompleteContext } from '@/components/v1/cloud/logging/log-search/autocomplete';
import { parseLogQuery } from '@/components/v1/cloud/logging/log-search/parser';
import type { AutocompleteSuggestion } from '@/components/v1/cloud/logging/log-search/types';
import { LOG_LEVEL_TO_API } from '@/components/v1/cloud/logging/log-search/types';
import { LogLine } from '@/components/v1/cloud/logging/log-search/use-logs';
import { LogViewer } from '@/components/v1/cloud/logging/log-viewer';
import { DocsButton } from '@/components/v1/docs/docs-button';
import { SearchBarWithFilters } from '@/components/v1/molecules/search-bar-with-filters/search-bar-with-filters';
import { OnboardingCard } from '@/components/v1/ui/onboarding-card';
import { useSidePanel } from '@/hooks/use-side-panel';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { V1LogLine, V1LogLineOrderByDirection } from '@/lib/api';
import api from '@/lib/api/api';
import { docsPages } from '@/lib/generated/docs';
import { useInfiniteQuery } from '@tanstack/react-query';
import { ScrollText } from 'lucide-react';
import { useCallback, useMemo, useState } from 'react';

const LOGS_PER_PAGE = 100;
const EMPTY_AUTOCOMPLETE_CONTEXT: LogAutocompleteContext = {};

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
      if (rows.length < LOGS_PER_PAGE) {
        return undefined;
      }
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
          defaultOpenTab: TabOption.Activity,
          showViewTaskRunButton: true,
        },
      });
    },
    [sidePanel],
  );

  const isEmpty = !logsQuery.isLoading && logs.length === 0;

  return (
    <div className="my-4 flex flex-col gap-y-2 max-h-[40rem] min-h-[25rem]">
      <SearchBarWithFilters<AutocompleteSuggestion, LogAutocompleteContext>
        value={queryString}
        onChange={setQueryString}
        onSubmit={setQueryString}
        getAutocomplete={(q, ctx) => getAutocomplete(q, ctx)}
        applySuggestion={applySuggestion}
        autocompleteContext={EMPTY_AUTOCOMPLETE_CONTEXT}
        placeholder="Search logs..."
        filterChips={[
          { key: 'level:', label: 'Level', description: 'Filter by log level' },
          {
            key: 'attempt:',
            label: 'Attempt',
            description: 'Filter by attempt number',
          },
        ]}
      />
      {isEmpty && (
        <OnboardingCard
          variant="info"
          icon={<ScrollText className="size-4" />}
          title="Send logs to Hatchet"
          dismissible
          dismissKey="hatchet:dismiss-logs-onboarding-hint"
          description="Configure Hatchet as a log sink to view your task logs for this run."
          actions={
            <DocsButton
              doc={docsPages.v1.logging}
              label="View logging docs"
              variant="text"
            />
          }
        />
      )}
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
