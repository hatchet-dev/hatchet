import { Waterfall } from '../../waterfall';
import { TabOption } from '../step-run-detail';
import { TaskRunTrace } from './task-run-trace';
import { Loading } from '@/components/v1/ui/loading';
import { useSidePanel } from '@/hooks/use-side-panel';
import api from '@/lib/api/api';
import { useQuery } from '@tanstack/react-query';
import { useCallback } from 'react';

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
    queryFn: async () => {
      const res = await api.v1TaskGetTrace(taskRunId);
      return res.data.rows || [];
    },
    refetchInterval: isRunning ? 1000 : false,
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
