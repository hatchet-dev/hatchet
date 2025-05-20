import { useEffect, useMemo, useState } from 'react';

import useDefinitions from '@/next/hooks/use-definitions';
import { Button } from '@/next/components/ui/button';
import { TriggerRunModal } from '@/next/components/runs/trigger-run-modal';
import { SignInRequiredAction } from './signin-required-action';
import { RunsProvider, useRuns } from '@/next/hooks/use-runs';
import { Loader2 } from 'lucide-react';
import { CheckCircle2 } from 'lucide-react';
import { ROUTES } from '@/next/lib/routes';
import { V1WorkflowRunDetails } from '@/lib/api';
import useTenant from '@/next/hooks/use-tenant';
interface TaskExecutionProps {
  name: string;
  input?: Record<string, unknown>;
  onRun?: (link: string) => void;
}

export function TaskExecution(props: TaskExecutionProps) {
  return (
    <RunsProvider
      initialTimeRange={{
        startTime: new Date().toISOString(),
      }}
      refetchInterval={1000}
    >
      <TaskExecutionContent {...props} />
    </RunsProvider>
  );
}

function TaskExecutionContent({ name, input, onRun }: TaskExecutionProps) {
  const { data: definitions } = useDefinitions();
  const [showTriggerModal, setShowTriggerModal] = useState(false);
  const {
    data: runs,
    filters: { setFilter },
  } = useRuns();
  const { tenant } = useTenant();

  const definitionId = useMemo(
    () => definitions?.find((d) => d.name === name)?.metadata.id,
    [definitions, name],
  );

  useEffect(() => {
    if (definitionId) {
      setFilter('workflow_ids', [definitionId]);
    }
  }, [definitionId, setFilter]);

  useEffect(() => {
    if (runs?.length > 0 && tenant?.metadata.id) {
      onRun?.(ROUTES.runs.detail(tenant?.metadata.id, runs[0].metadata.id));
    }
  }, [runs, onRun]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-4">
        <SignInRequiredAction>
          <div className="flex items-center gap-4 p-4 bg-muted rounded-lg">
            <Button onClick={() => setShowTriggerModal(true)}>Run Task</Button>

            <div className="flex items-center gap-2">
              {runs?.length === 0 ? (
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  <span>
                    Waiting for <pre className="inline">{name}</pre> to run...
                  </span>
                </div>
              ) : (
                <>
                  <CheckCircle2 className="h-4 w-4 text-green-500" />
                  <span>Task triggered successfully!</span>
                </>
              )}
            </div>
          </div>
        </SignInRequiredAction>
      </div>
      <TriggerRunModal
        show={showTriggerModal}
        onClose={() => setShowTriggerModal(false)}
        defaultWorkflowId={definitionId}
        defaultInput={JSON.stringify(input)}
        disabledCapabilities={['timing', 'fromRecent', 'additionalMeta']}
        onRun={(run) => {
          if (tenant?.metadata.id) {
            onRun?.(
              ROUTES.runs.detail(
                tenant.metadata.id,
                (run as V1WorkflowRunDetails).run.metadata.id,
              ),
            );
          }
          setShowTriggerModal(false);
        }}
      />
    </div>
  );
}
