import { ScheduledWorkflows } from '@/lib/api';
import { Separator } from '@/components/v1/ui/separator';
import RelativeDate from '@/components/v1/molecules/relative-date';
import { CodeHighlighter } from '@/components/v1/ui/code-highlighter';
import { RunStatus } from '../../../workflow-runs/components/run-statuses';
import { Link } from 'react-router-dom';
import { useCurrentTenantId } from '@/hooks/use-tenant';

export function ExpandedScheduledRunContent({
  scheduledRun,
}: {
  scheduledRun: ScheduledWorkflows;
}) {
  const { tenantId } = useCurrentTenantId();

  return (
    <div className="w-full">
      <div className="space-y-6">
        <div className="flex flex-col justify-center items-start gap-3 pb-4 border-b text-sm">
          <div className="flex flex-row items-center gap-3 min-w-0 w-full">
            <span className="text-muted-foreground font-medium shrink-0">
              Workflow
            </span>
            <Link
              to={`/tenants/${tenantId}/workflows/${scheduledRun.workflowId}`}
              className="px-2 py-1 overflow-x-auto min-w-0 flex-1 hover:underline"
            >
              <span className="whitespace-nowrap">
                {scheduledRun.workflowName}
              </span>
            </Link>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground font-medium">
              Trigger At
            </span>
            <span className="font-medium">
              <RelativeDate date={scheduledRun.triggerAt} />
            </span>
          </div>
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground font-medium">Status</span>
            <RunStatus status={scheduledRun.workflowRunStatus || 'SCHEDULED'} />
          </div>
          {scheduledRun.workflowRunId && (
            <div className="flex flex-row items-center gap-3 min-w-0 w-full">
              <span className="text-muted-foreground font-medium shrink-0">
                Workflow Run
              </span>
              <Link
                to={`/tenants/${tenantId}/runs/${scheduledRun.workflowRunId}`}
                className="px-2 py-1 overflow-x-auto min-w-0 flex-1 hover:underline"
              >
                <span className="whitespace-nowrap">
                  {scheduledRun.workflowRunName || scheduledRun.workflowRunId}
                </span>
              </Link>
            </div>
          )}
          <div className="flex items-center gap-3">
            <span className="text-muted-foreground font-medium">
              Created At
            </span>
            <span className="font-medium">
              <RelativeDate date={scheduledRun.metadata.createdAt} />
            </span>
          </div>
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
