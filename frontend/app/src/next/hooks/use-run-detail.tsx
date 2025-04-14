import { useQuery } from '@tanstack/react-query';
import api from '@/next/lib/api';
import { useState } from 'react';
import { useRuns } from '@/next/hooks/use-runs';

export function useRunDetail(runId: string, defaultRefetchInterval?: number) {
  const { cancel, replay } = useRuns({});

  const [refetchInterval, setRefetchInterval] = useState(
    defaultRefetchInterval,
  );

  const runDetails = useQuery({
    queryKey: ['workflow-run-details:get', runId],
    queryFn: async () => {
      if (runId == '00000000-0000-0000-0000-000000000000') {
        return;
      }
      const run = (await api.v1WorkflowRunGet(runId)).data;

      if (refetchInterval) {
        setRefetchInterval(
          run.run.status === 'RUNNING' ? 1000 : defaultRefetchInterval || 0,
        );
      }

      return run;
    },
    refetchInterval: refetchInterval,
  });

  const activity = useQuery({
    queryKey: ['task-events:get', runId, runDetails.data?.tasks],
    queryFn: async () => {
      if (runId == '00000000-0000-0000-0000-000000000000') {
        return;
      }

      const tasks = runDetails.data?.tasks || [];

      const logPromises = tasks.map(async (task) =>
        ((await api.v1LogLineList(task.metadata.id)).data?.rows || []).map(
          (log) => ({
            ...log,
            taskId: task.metadata.id,
          }),
        ),
      );

      const eventPromises = tasks.map(
        async (task) =>
          (
            await api.v1TaskEventList(task.metadata.id, {
              limit: 50,
              offset: 0,
            })
          ).data?.rows || [],
      );

      const [logs, events] = await Promise.all([
        Promise.all(logPromises),
        Promise.all(eventPromises),
      ]);

      const mergedLogs = logs.flat();
      const mergedEvents = events.flat();

      return { events: mergedEvents, logs: mergedLogs };
    },
    refetchInterval: refetchInterval,
  });

  return {
    ...runDetails,
    activity: activity.data,
    cancel,
    replay,
  };
}
