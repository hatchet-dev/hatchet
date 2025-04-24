import { V1TaskSummary, V1WorkflowRun, V1WorkflowType } from '@/lib/api';
import { Duration } from 'date-fns';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Link, useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';

export interface RunIdProps {
  wfRun?: V1WorkflowRun;
  taskRun?: V1TaskSummary;
  onClick?: () => void;
}

export function RunId({ wfRun, taskRun, onClick }: RunIdProps) {
  const isTaskRun = taskRun !== undefined;
  const navigate = useNavigate();

  if (taskRun?.displayName.startsWith('leaf')) {
    // Debugging code removed.
  }

  const url = !isTaskRun
    ? ROUTES.runs.detail(wfRun?.metadata.id || '')
    : taskRun?.type == V1WorkflowType.TASK
      ? undefined
      : ROUTES.runs.taskDetail(
          taskRun?.workflowRunExternalId || '',
          taskRun?.taskExternalId || '',
        );

  const name = isTaskRun
    ? getFriendlyTaskRunId(taskRun)
    : getFriendlyWorkflowRunId(wfRun);

  const handleDoubleClick = () => {
    if (url) {
      navigate(url);
    }
  };

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            {url && !onClick ? (
              <Link to={url} className="hover:underline text-foreground">
                {name}
              </Link>
            ) : (
              <span
                className={onClick ? 'cursor-pointer' : ''}
                onClick={onClick}
                onDoubleClick={handleDoubleClick}
              >
                {name}
              </span>
            )}
          </span>
        </TooltipTrigger>
        <TooltipContent className="bg-muted">
          <div className="font-mono text-foreground">
            {wfRun?.metadata.id || taskRun?.metadata.id || ''}
          </div>
        </TooltipContent>
      </Tooltip>
    </TooltipProvider>
  );
}

export function splitTime(runId?: string) {
  if (!runId) {
    return;
  }

  return runId.split('-').slice(0, -1).join('-');
}

export function getFriendlyTaskRunId(run?: V1TaskSummary) {
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

export function formatDuration(duration: Duration, rawTimeMs: number): string {
  const parts = [];

  if (duration.days) {
    parts.push(`${duration.days}d`);
  }

  if (duration.hours) {
    parts.push(`${duration.hours}h`);
  }

  if (duration.minutes) {
    parts.push(`${duration.minutes}m`);
  }

  if (rawTimeMs < 10000 && duration.seconds) {
    const ms = Math.floor((rawTimeMs % 1000) / 10);
    parts.push(`${duration.seconds}.${ms.toString().padStart(2, '0')}s`);
    return parts.join(' ');
  }

  if (duration.seconds) {
    parts.push(`${duration.seconds}s`);
  }

  if (rawTimeMs < 1000) {
    const ms = rawTimeMs % 1000;
    parts.push(`${ms}ms`);
  }

  return parts.join(' ');
}
