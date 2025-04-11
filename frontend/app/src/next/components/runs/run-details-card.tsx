import { Clock } from 'lucide-react';
import { Card, CardContent, CardHeader, CardTitle } from '@/next/components/ui/card';
import { Time } from '@/next/components/ui/time';
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from '@/next/components/ui/tooltip';
import { intervalToDuration } from 'date-fns';
import { Code } from '@/next/components/ui/code';
import { useRunDetail } from '@/next/hooks/use-run-detail';
import { Skeleton } from '../ui/skeleton';
import { RunsBadge } from './runs-badge';

export interface RunDetailsCardProps {
  runId: string;
}

export function RunDetailsCard({ runId }: RunDetailsCardProps) {
  const { data, isLoading, error } = useRunDetail(runId || '');

  const run = data?.run;

  if (isLoading) {
    return <Skeleton className="h-[100px] w-full" />;
  }

  if (error || !run) {
    return <div>Error loading run details</div>;
  }

  // Calculate duration
  const startedAt = run.startedAt ? new Date(run.startedAt) : null;
  const finishedAt = run.finishedAt ? new Date(run.finishedAt) : null;
  const isRunning = run.status === 'RUNNING';

  return (
    <Card className="overflow-hidden">
      <CardHeader className="py-3 px-4">
        <CardTitle className="text-sm font-medium">Run Details</CardTitle>
      </CardHeader>
      <CardContent className="p-0">
        <div className="grid grid-cols-3 text-sm divide-y">
          {/* ID */}
          <div className="col-span-1 font-medium bg-muted/30 px-4 py-1.5 border-r">
            ID
          </div>
          <div className="col-span-2 px-4 py-1.5">
            <Code
              variant="inline"
              className="font-mono text-xs"
              language={'plaintext'}
              value={run.metadata.id}
            >
              {run.metadata.id}
            </Code>
          </div>

          {/* Workflow */}
          <div className="col-span-1 font-medium bg-muted/30 px-4 py-1.5 border-r">
            Workflow
          </div>
          <div className="col-span-2 px-4 py-1.5">{run.workflowId}</div>

          {/* Status */}
          <div className="col-span-1 font-medium bg-muted/30 px-4 py-1.5 border-r">
            Status
          </div>
          <div className="col-span-2 px-4 py-1.5">
            <RunsBadge status={run.status} variant="default" />
          </div>

          {/* Created */}
          <div className="col-span-1 font-medium bg-muted/30 px-4 py-1.5 border-r">
            Created
          </div>
          <div className="col-span-2 px-4 py-1.5">
            {run.createdAt ? (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>
                      <Time
                        date={run.createdAt}
                        variant="timeSince"
                        className="text-muted-foreground"
                      />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <Time date={run.createdAt} variant="timestamp" />
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ) : (
              <span className="text-muted-foreground">-</span>
            )}
          </div>

          {/* Started */}
          <div className="col-span-1 font-medium bg-muted/30 px-4 py-1.5 border-r">
            Started
          </div>
          <div className="col-span-2 px-4 py-1.5">
            {run.startedAt ? (
              <TooltipProvider>
                <Tooltip>
                  <TooltipTrigger asChild>
                    <span>
                      <Time
                        date={run.startedAt}
                        variant="timeSince"
                        className="text-muted-foreground"
                      />
                    </span>
                  </TooltipTrigger>
                  <TooltipContent>
                    <Time date={run.startedAt} variant="timestamp" />
                  </TooltipContent>
                </Tooltip>
              </TooltipProvider>
            ) : (
              <span className="text-muted-foreground">Not started</span>
            )}
          </div>

          {/* Duration */}
          <div className="col-span-1 font-medium bg-muted/30 px-4 py-1.5 border-r">
            Duration
          </div>
          <div className="col-span-2 px-4 py-1.5">
            <div className="flex items-center gap-1">
              <Clock className="h-3.5 w-3.5 text-muted-foreground" />
              <span className={isRunning ? 'animate-pulse' : ''}>
                {startedAt ? (
                  <>
                    {(() => {
                      const start = startedAt;
                      const end = finishedAt || new Date();
                      const duration = intervalToDuration({ start, end });

                      // Use compact duration format from columns.tsx
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
                      if (duration.seconds || !parts.length) {
                        parts.push(`${duration.seconds || 0}s`);
                      }

                      return parts.length ? parts.join(' ') : '< 1s';
                    })()}
                    {isRunning && '...'}
                  </>
                ) : (
                  'Not started'
                )}
              </span>
            </div>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
