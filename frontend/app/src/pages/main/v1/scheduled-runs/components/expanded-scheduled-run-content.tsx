import { RunStatus } from '../../workflow-runs/components/run-statuses';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { Separator } from '@/components/v1/ui/separator';
import { useCurrentTenantId } from '@/hooks/use-tenant';
import { ScheduledWorkflows } from '@/lib/api';
import { appRoutes } from '@/router';
import { Link } from '@tanstack/react-router';

export function ExpandedScheduledRunContent({
  scheduledRun,
}: {
  scheduledRun: ScheduledWorkflows;
}) {
  const { tenantId } = useCurrentTenantId();

  return (
    <div className="w-full">
      <div className="space-y-6">
        <div className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-3 border-b pb-4 text-sm">
          <span className="font-medium text-muted-foreground">Workflow</span>
          <Link
            to={appRoutes.tenantWorkflowRoute.to}
            params={{ tenant: tenantId, workflow: scheduledRun.workflowId }}
            className="truncate font-medium hover:underline"
          >
            {scheduledRun.workflowName}
          </Link>

          <span className="font-medium text-muted-foreground">Trigger At</span>
          <span className="font-medium">
            <RelativeDate date={scheduledRun.triggerAt} />
          </span>

          <span className="font-medium text-muted-foreground">Status</span>
          <div>
            <RunStatus status={scheduledRun.workflowRunStatus || 'SCHEDULED'} />
          </div>

          {scheduledRun.workflowRunId && (
            <>
              <span className="font-medium text-muted-foreground">
                Workflow Run
              </span>
              <Link
                to={appRoutes.tenantRunRoute.to}
                params={{ tenant: tenantId, run: scheduledRun.workflowRunId }}
                className="truncate font-medium hover:underline"
              >
                {scheduledRun.workflowRunName || scheduledRun.workflowRunId}
              </Link>
            </>
          )}

          <span className="font-medium text-muted-foreground">Created At</span>
          <span className="font-medium">
            <RelativeDate date={scheduledRun.metadata.createdAt} />
          </span>
        </div>

        <div className="space-y-4">
          {scheduledRun.input && (
            <div>
              <h3 className="mb-2 text-sm font-semibold text-foreground">
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
                <h3 className="mb-2 text-sm font-semibold text-foreground">
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
