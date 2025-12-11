import { ScheduledWorkflows } from '@/lib/api';
import { Separator } from '@/components/v1/ui/separator';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Link } from '@tanstack/react-router';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { RunStatus } from '../../workflow-runs/components/run-statuses';
import { appRoutes } from '@/router';

export function ExpandedScheduledRunContent({
  scheduledRun,
}: {
  scheduledRun: ScheduledWorkflows;
}) {
  const { tenantId } = useCurrentTenantId();

  return (
    <div className="w-full">
      <div className="space-y-6">
        <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 pb-4 border-b text-sm">
          <span className="text-muted-foreground font-medium">Workflow</span>
          <Link
            to={appRoutes.tenantWorkflowRoute.to}
            params={{ tenant: tenantId, workflow: scheduledRun.workflowId }}
            className="font-medium hover:underline truncate"
          >
            {scheduledRun.workflowName}
          </Link>

          <span className="text-muted-foreground font-medium">Trigger At</span>
          <span className="font-medium">
            <RelativeDate date={scheduledRun.triggerAt} />
          </span>

          <span className="text-muted-foreground font-medium">Status</span>
          <div>
            <RunStatus status={scheduledRun.workflowRunStatus || 'SCHEDULED'} />
          </div>

          {scheduledRun.workflowRunId && (
            <>
              <span className="text-muted-foreground font-medium">
                Workflow Run
              </span>
              <Link
                to="/tenants/$tenant/runs/$run"
                params={{ tenant: tenantId, run: scheduledRun.workflowRunId }}
                className="font-medium hover:underline truncate"
              >
                {scheduledRun.workflowRunName || scheduledRun.workflowRunId}
              </Link>
            </>
          )}

          <span className="text-muted-foreground font-medium">Created At</span>
          <span className="font-medium">
            <RelativeDate date={scheduledRun.metadata.createdAt} />
          </span>
        </div>

        <div className="space-y-4">
          {scheduledRun.input && (
            <div>
              <h3 className="text-sm font-semibold text-foreground mb-2">
                Payload
              </h3>
              <Separator className="mb-3" />
              <div className="max-h-96 overflow-y-auto rounded-lg">
                <CodeHighlighter
                  language="json"
                  className="text-xs"
                  code={JSON.stringify(scheduledRun.input, null, 2)}
                />
              </div>
            </div>
          )}

          {scheduledRun.additionalMetadata &&
            Object.keys(scheduledRun.additionalMetadata).length > 0 && (
              <div>
                <h3 className="text-sm font-semibold text-foreground mb-2">
                  Metadata
                </h3>
                <Separator className="mb-3" />
                <div className="max-h-96 overflow-y-auto rounded-lg">
                  <CodeHighlighter
                    language="json"
                    className="text-xs"
                    code={JSON.stringify(
                      scheduledRun.additionalMetadata,
                      null,
                      2,
                    )}
                  />
                </div>
              </div>
            )}
        </div>
      </div>
    </div>
  );
}
