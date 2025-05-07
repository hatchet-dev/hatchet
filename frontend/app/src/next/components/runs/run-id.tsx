import { V1TaskSummary, V1WorkflowRun, V1WorkflowType } from '@/lib/api';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { Link, useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';

interface RunIdProps {
  wfRun?: V1WorkflowRun;
  taskRun?: V1TaskSummary;
  displayName?: string;
  id?: string;
  onClick?: () => void;
  noLink?: boolean;
}

export function RunId({ wfRun, taskRun, displayName, id, onClick, noLink }: RunIdProps) {
  const isTaskRun = taskRun !== undefined;
  const navigate = useNavigate();

  const url = !isTaskRun
    ? ROUTES.runs.detail(wfRun?.metadata.id || id || '')
    : taskRun?.type == V1WorkflowType.TASK
      ? undefined
      : ROUTES.runs.taskDetail(
          taskRun?.workflowRunExternalId || '',
          taskRun?.taskExternalId || '',
        );

  const name = isTaskRun
    ? getFriendlyTaskRunId(taskRun)
    : displayName && id
      ? splitTime(displayName) + '-' + id.split('-')[0]
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
            {url && !onClick && !noLink ? (
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
            {wfRun?.metadata.id || taskRun?.metadata.id || id || ''}
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
