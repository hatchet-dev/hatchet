import { V1TaskSummary, V1WorkflowRun, V1WorkflowType } from '@/lib/api';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';
import { cn } from '@/lib/utils';
import { useCurrentTenantId } from '@/next/hooks/use-tenant';

interface RunIdProps {
  wfRun?: V1WorkflowRun;
  taskRun?: V1TaskSummary;
  displayName?: string;
  id?: string;
  onClick?: () => void;
  className?: string;
  attempt?: number;
}

export function RunId({
  wfRun,
  taskRun,
  displayName,
  id,
  className,
  onClick,
  attempt,
}: RunIdProps) {
  const isTaskRun = taskRun !== undefined;
  const navigate = useNavigate();
  const { tenantId } = useCurrentTenantId();

  const url = !isTaskRun
    ? ROUTES.runs.detail(tenantId, wfRun?.metadata.id || id || '')
    : taskRun?.type === V1WorkflowType.TASK
      ? undefined
      : ROUTES.runs.detail(tenantId, taskRun?.workflowRunExternalId || '');

  const displayNameIdPrefix = splitTime(displayName);
  const friendlyDisplayName = displayNameIdPrefix || displayName;

  const name = isTaskRun
    ? getFriendlyTaskRunId(taskRun)
    : displayName && id
      ? friendlyDisplayName + '-' + id.split('-')[0]
      : getFriendlyWorkflowRunId(wfRun);

  const handleDoubleClick = () => {
    if (url && !onClick) {
      navigate(url);
    }
  };

  const displayAttempt = attempt || taskRun?.attempt;

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            {url && !onClick ? (
              <span
                className={cn(
                  'hover:underline text-foreground cursor-pointer',
                  className,
                )}
                onClick={() => navigate(url)}
              >
                {name}
              </span>
            ) : (
              <span
                className={cn(onClick ? 'cursor-pointer' : '', className)}
                onClick={onClick}
                onDoubleClick={handleDoubleClick}
              >
                {name}
                {displayAttempt !== undefined ? `/${displayAttempt}` : null}
              </span>
            )}
          </span>
        </TooltipTrigger>
        <TooltipContent className="bg-muted">
          <div className="font-mono text-foreground">
            Run Id: {wfRun?.metadata.id || taskRun?.metadata.id || id || ''}
            <br />
            {displayAttempt ? `Attempt: ${displayAttempt}` : null}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

function splitTime(runId?: string) {
  if (!runId) {
    return;
  }

  return runId.split('-').slice(0, -1).join('-');
}

function getFriendlyTaskRunId(run?: V1TaskSummary) {
  if (!run) {
    return;
  }

  if (run.actionId) {
    const runIdPrefix = run.metadata.id.split('-')[0];
    return run.actionId?.split(':')?.at(1) + '-' + runIdPrefix;
  }

  return getFriendlyWorkflowRunId(run);
}

export function getFriendlyWorkflowRunId(run?: V1WorkflowRun) {
  if (!run) {
    return;
  }

  const displayNameParts = splitTime(run.displayName);
  const runIdPrefix = run.metadata.id.split('-')[0];

  return displayNameParts + '-' + runIdPrefix;
}
