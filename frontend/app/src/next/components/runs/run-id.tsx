import { V1TaskSummary, V1WorkflowRun, V1WorkflowType } from '@/next/lib/api';
import { Duration } from 'date-fns';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Code } from '@/next/components/ui/code';
import { useMemo } from 'react';
import { Link } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';

export function RunId({
  wfRun,
  taskRun,
}: {
  wfRun?: V1WorkflowRun;
  taskRun?: V1TaskSummary;
}) {
  const isTaskRun = taskRun !== undefined;

  if (taskRun?.displayName.startsWith('leaf')) {
    console.log(taskRun);
  }

  const url = !isTaskRun
    ? ROUTES.runs.detail(wfRun?.metadata.id || '')
    : taskRun?.type == V1WorkflowType.TASK
      ? undefined
      : ROUTES.runs.detail(taskRun?.taskExternalId || '');

  const name = useMemo(() => {
    if (isTaskRun) {
      return getFriendlyTaskRunId(taskRun);
    }

    return getFriendlyWorkflowRunId(wfRun);
  }, [isTaskRun, taskRun, wfRun]);

  return (
    <TooltipProvider>
      <Tooltip>
        <TooltipTrigger asChild>
          <span>
            {url ? (
              <Link to={url} className="hover:underline text-blue-500">
                {name}
              </Link>
            ) : (
              name
            )}
          </span>
        </TooltipTrigger>
        <TooltipContent>
          <Code
            variant="inline"
            className="font-medium"
            language={'plaintext'}
            value={wfRun?.metadata.id || taskRun?.metadata.id || ''}
          >
            {wfRun?.metadata.id || taskRun?.metadata.id || ''}
          </Code>
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

  const [first, second] = run.actionId.split(':');
  const runIdPrefix = run.metadata.id.split('-')[0];

  return run.actionId
    ? first === second
      ? first + '-' + runIdPrefix
      : run.actionId + '-' + runIdPrefix
    : getFriendlyWorkflowRunId(run);
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
