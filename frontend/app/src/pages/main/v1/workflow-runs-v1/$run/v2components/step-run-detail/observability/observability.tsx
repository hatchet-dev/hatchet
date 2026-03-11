import { Waterfall } from '../../waterfall';
import { TabOption } from '../step-run-detail';
import { TaskRunTrace } from './task-run-trace';
import { Loading } from '@/components/v1/ui/loading';
import { useSidePanel } from '@/hooks/use-side-panel';
import api from '@/lib/api/api';
import { OtelSpan } from '@/lib/api/generated/data-contracts';
import { useQuery } from '@tanstack/react-query';
import { useCallback } from 'react';

const PAGE_SIZE = 200;

async function fetchAllSpans(taskExternalId: string): Promise<OtelSpan[]> {
  const allSpans: OtelSpan[] = [];
  let offset = 0;

  // eslint-disable-next-line no-constant-condition
  while (true) {
    const res = await api.v1TaskGetTrace(taskExternalId, {
      offset,
      limit: PAGE_SIZE,
    });

    const rows = res.data.rows ?? [];
    allSpans.push(...rows);

    const numPages = res.data.pagination?.num_pages ?? 1;
    const currentPage = res.data.pagination?.current_page ?? 1;

    if (currentPage >= numPages || rows.length === 0) {
      break;
    }

    offset += PAGE_SIZE;
  }

  return allSpans;
}

export const Observability = ({
  taskRunId,
  isRunning,
}: {
  taskRunId: string;
  isRunning: boolean;
}) => {
  const { open } = useSidePanel();

  const handleTaskRunExpand = useCallback(
    (taskRunId: string) => {
      open({
        type: 'task-run-details',
        content: {
          taskRunId,
          defaultOpenTab: TabOption.Output,
          showViewTaskRunButton: true,
        },
      });
    },
    [open],
  );

  const tracesQuery = useQuery({
    queryKey: ['task:trace', taskRunId],
    queryFn: () => fetchAllSpans(taskRunId),
    refetchInterval: isRunning ? 5000 : false,
  });

  if (!tracesQuery.isFetched) {
    return <Loading />;
  }

  if (!tracesQuery.data || tracesQuery.data.length === 0) {
    return (
      <Waterfall
        workflowRunId={taskRunId}
        selectedTaskId={undefined}
        handleTaskSelect={handleTaskRunExpand}
      />
    );
  }

  return <TaskRunTrace spans={tracesQuery.data} taskRunId={taskRunId} />;
};
