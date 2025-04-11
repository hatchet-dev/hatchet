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

  const taskEvents = useQuery({
    queryKey: ['task-events:get', runId],
    queryFn: async () => {
      const events = (
        await api.v1TaskEventList(runId, {
          limit: 50,
          offset: 0,
        })
      ).data;

      return events;
    },
    refetchInterval: refetchInterval,
  });

  return {
    ...runDetails,
    taskEvents,
    cancel,
    replay,
  };
}
