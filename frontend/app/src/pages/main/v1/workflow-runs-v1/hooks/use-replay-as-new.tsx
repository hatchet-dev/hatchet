import { TriggerWorkflowForm } from '../../workflows/$workflow/components/trigger-workflow-form';
import { useToast } from '@/components/v1/hooks/use-toast';
import useCanViewPayloads from '@/hooks/use-can-view-payloads';
import useCanWrite from '@/hooks/use-can-write';
import { queries, V1TaskSummary, V1WorkflowType } from '@/lib/api';
import { isStandaloneRun } from '@/lib/task-runs';
import { useQueryClient } from '@tanstack/react-query';
import { useCallback, useState } from 'react';

type ReplayAsNewRun = Pick<
  V1TaskSummary,
  'metadata' | 'type' | 'workflowId' | 'additionalMetadata'
>;

export function replayAsNewDisabledReason(
  run: Pick<V1TaskSummary, 'metadata' | 'workflowRunExternalId'>,
): string | undefined {
  return isStandaloneRun(run)
    ? undefined
    : 'Only top-level runs can be replayed as new. Use the parent run instead.';
}

export function useReplayAsNew() {
  const queryClient = useQueryClient();
  const { toast } = useToast();
  const canViewPayloads = useCanViewPayloads();
  const canWrite = useCanWrite();
  const [run, setRun] = useState<
    (ReplayAsNewRun & Pick<V1TaskSummary, 'input'>) | null
  >(null);

  const replayAsNew = useCallback(
    async (target: ReplayAsNewRun) => {
      try {
        const { input, payloadsRestricted } =
          target.type === V1WorkflowType.DAG
            ? (
                await queryClient.fetchQuery(
                  queries.v1WorkflowRuns.details(target.metadata.id),
                )
              ).run
            : await queryClient.fetchQuery(
                queries.v1Tasks.get(target.metadata.id),
              );

        if (payloadsRestricted) {
          toast({
            title: 'You do not have permission to view this run input',
            variant: 'destructive',
          });
          return;
        }

        setRun({ ...target, input });
      } catch {
        toast({
          title: 'Failed to load run input',
          variant: 'destructive',
        });
      }
    },
    [queryClient, toast],
  );

  const dialog = run ? (
    <TriggerWorkflowForm
      key={run.metadata.id}
      defaultWorkflowId={run.workflowId}
      defaultInput={JSON.stringify(run.input ?? {}, null, 2)}
      defaultAddlMeta={JSON.stringify(run.additionalMetadata ?? {}, null, 2)}
      show
      onClose={() => setRun(null)}
    />
  ) : null;

  return {
    canReplayAsNew: canViewPayloads && canWrite,
    replayAsNew,
    dialog,
  };
}
