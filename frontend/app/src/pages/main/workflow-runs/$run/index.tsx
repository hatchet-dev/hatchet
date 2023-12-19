import { Icons } from '@/components/ui/icons';
import { Separator } from '@/components/ui/separator';
import { JobRun, StepRun, StepRunStatus, queries, Event } from '@/lib/api';
import CronPrettifier from 'cronstrue';
import { currTenantAtom } from '@/lib/atoms';
import { useQuery } from '@tanstack/react-query';
import { useAtom } from 'jotai';
import { Link, useParams } from 'react-router-dom';
import invariant from 'tiny-invariant';
import { Badge } from '@/components/ui/badge';
import { relativeDate } from '@/lib/utils';
import {
  AdjustmentsHorizontalIcon,
  BoltIcon,
  Square3Stack3DIcon,
} from '@heroicons/react/24/outline';
import { Button } from '@/components/ui/button';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { JobRunColumns, columns } from './components/job-runs-columns';
import { TableCell, TableRow } from '@/components/ui/table';
import { RunStatus } from '../components/run-statuses';
import { ColumnDef } from '@tanstack/react-table';
import { useState } from 'react';
import { Code } from '@/components/ui/code';

export default function ExpandedWorkflowRun() {
  const [expandedStepRuns, setExpandedStepRuns] = useState<string[]>([]);

  const [tenant] = useAtom(currTenantAtom);
  invariant(tenant);

  const params = useParams();
  invariant(params.run);

  const runQuery = useQuery({
    ...queries.workflowRuns.get(tenant.metadata.id, params.run),
  });

  if (runQuery.isLoading || !runQuery.data) {
    return (
      <div className="flex flex-row flex-1 w-full h-full">
        <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
      </div>
    );
  }

  const run = runQuery.data;

  return (
    <div className="flex-grow h-full w-full">
      <div className="mx-auto max-w-7xl py-8 px-4 sm:px-6 lg:px-8">
        <div className="flex flex-row justify-between items-center">
          <div className="flex flex-row gap-4 items-center">
            <AdjustmentsHorizontalIcon className="h-6 w-6 text-foreground mt-1" />
            <h2 className="text-2xl font-bold leading-tight text-foreground">
              {run?.metadata.id}
            </h2>
            <Badge className="text-sm mt-1" variant={'secondary'}>
              {/* {workflow.versions && workflow.versions[0].version} */}
              {run.status}
            </Badge>
          </div>
        </div>
        <div className="flex flex-row justify-start items-center mt-4 gap-2">
          {run?.workflowVersion?.workflow && (
            <Link
              to={`/workflows/${run?.workflowVersion?.workflow?.metadata.id}`}
            >
              <Button
                variant="ghost"
                className="flex flex-row items-center gap-2 text-sm text-foreground hover:bg-muted"
              >
                <Square3Stack3DIcon className="h-4 w-4" />
                {run?.workflowVersion?.workflow?.name}
              </Button>
            </Link>
          )}
          <div className="text-sm text-muted-foreground">
            Created {relativeDate(run?.metadata.createdAt)}
          </div>
          {run?.startedAt && (
            <div className="text-sm text-muted-foreground">
              Started {relativeDate(run?.startedAt)}
            </div>
          )}
          {run?.finishedAt && (
            <div className="text-sm text-muted-foreground">
              Finished {relativeDate(run?.startedAt)}
            </div>
          )}
        </div>
        <Separator className="my-4" />
        {run.triggeredBy?.event && (
          <TriggeringEventSection event={run.triggeredBy.event} />
        )}
        {run.triggeredBy?.cronSchedule && (
          <TriggeringCronSection cron={run.triggeredBy.cronSchedule} />
        )}
        <h3 className="text-xl font-bold leading-tight text-foreground mb-4">
          Job Runs
        </h3>
        <DataTable
          columns={columns}
          data={
            run.jobRuns
              ?.map((jobRun): JobRunColumns[] => {
                return [
                  {
                    kind: 'job',
                    isExpandable: false,
                    getRow: () => {
                      return getJobRunRow({ jobRun, columns });
                    },
                    ...jobRun,
                  },
                  ...(jobRun.stepRuns
                    ?.map((stepRun): JobRunColumns[] => {
                      const res: JobRunColumns[] = [
                        {
                          kind: 'step',
                          isExpandable: true,
                          onClick: () => {
                            if (
                              expandedStepRuns.includes(stepRun.metadata.id)
                            ) {
                              setExpandedStepRuns(
                                expandedStepRuns.filter(
                                  (id) => id != stepRun.metadata.id,
                                ),
                              );
                            } else {
                              setExpandedStepRuns([
                                ...expandedStepRuns,
                                stepRun.metadata.id,
                              ]);
                            }
                          },
                          ...stepRun,
                        },
                      ];

                      if (expandedStepRuns.includes(stepRun.metadata.id)) {
                        res.push({
                          kind: 'step',
                          isExpandable: false,

                          getRow: () => {
                            return getExpandedStepRunRow({ stepRun, columns });
                          },
                          ...stepRun,
                        });
                      }

                      return res;
                    })
                    .flat() || []),
                ];
              })
              .flat() || []
          }
          filters={[]}
        />
      </div>
    </div>
  );
}

function getJobRunRow({
  jobRun,
}: {
  jobRun: JobRun;
  columns: ColumnDef<JobRunColumns>[];
}) {
  return (
    <TableRow key={jobRun.metadata.id} className="bg-muted">
      <TableCell colSpan={1}>
        <div className="flex flex-row gap-2 items-center justify-start">
          <BoltIcon className="h-4 w-4" />
          {jobRun.job?.name}
        </div>
      </TableCell>
      <TableCell colSpan={1}>
        <RunStatus status={jobRun.status} />
      </TableCell>
      <TableCell colSpan={1}>
        <div>{relativeDate(jobRun.startedAt)}</div>
      </TableCell>
      <TableCell colSpan={1}>
        <div>{relativeDate(jobRun.finishedAt)}</div>
      </TableCell>
    </TableRow>
  );
}

function getExpandedStepRunRow({
  stepRun,
  columns,
}: {
  stepRun: StepRun;
  columns: ColumnDef<JobRunColumns>[];
}) {
  return (
    <TableRow key={stepRun.metadata.id}>
      <TableCell colSpan={columns.length} className="px-8 py-4">
        <StepStatusSection stepRun={stepRun} />
        <StepConfigurationSection stepRun={stepRun} />
        <StepInputSection stepRun={stepRun} />
        <StepOutputSection stepRun={stepRun} />
      </TableCell>
    </TableRow>
  );
}

function StepInputSection({ stepRun }: { stepRun: StepRun }) {
  const input = stepRun.input || '{}';

  return (
    <>
      <h3 className="font-semibold leading-tight text-foreground mb-4">
        Input
      </h3>
      <Code language="json" className="my-4" maxHeight="400px">
        {JSON.stringify(JSON.parse(input), null, 2)}
      </Code>
    </>
  );
}

function StepOutputSection({ stepRun }: { stepRun: StepRun }) {
  const output = stepRun.output || '{}';

  return (
    <>
      <h3 className="font-semibold leading-tight text-foreground mb-4">
        Output
      </h3>
      <Code language="json" className="my-4" maxHeight="400px">
        {JSON.stringify(JSON.parse(output), null, 2)}
      </Code>
    </>
  );
}

function StepStatusSection({ stepRun }: { stepRun: StepRun }) {
  let statusText = 'Unknown';

  switch (stepRun.status) {
    case StepRunStatus.RUNNING:
      statusText = 'This step is currently running';
      break;
    case StepRunStatus.FAILED:
      statusText = 'This step failed';

      if (stepRun.error) {
        statusText = `This step failed with error ${stepRun.error}`;
      }

      break;
    case StepRunStatus.CANCELLED:
      statusText = 'This step was cancelled';

      switch (stepRun.cancelledReason) {
        case 'TIMED_OUT':
          statusText = `This step was cancelled because it exceeded its timeout of ${
            stepRun.step?.timeout || '60s'
          }`;
          break;
        case 'PREVIOUS_STEP_TIMED_OUT':
          statusText =
            'This step was cancelled because the previous step timed out';
          break;
        default:
          break;
      }

      break;
    case StepRunStatus.SUCCEEDED:
      statusText = 'This step succeeded';
      break;
    case StepRunStatus.PENDING:
      statusText = 'This step is pending';
      break;
    default:
      break;
  }

  return (
    <div className="mb-4">
      <h3 className="font-semibold leading-tight text-foreground mb-4">
        Status
      </h3>
      <div className="text-sm text-muted-foreground">{statusText}</div>
    </div>
  );
}

function StepConfigurationSection({ stepRun }: { stepRun: StepRun }) {
  return (
    <div className="mb-4">
      <h3 className="font-semibold leading-tight text-foreground mb-4">
        Configuration
      </h3>
      <div className="text-sm text-muted-foreground">
        Timeout: {stepRun.step?.timeout || '60s'}
      </div>
    </div>
  );
}

function TriggeringEventSection({ event }: { event: Event }) {
  return (
    <>
      <h3 className="text-xl font-semibold leading-tight text-foreground mb-4">
        Triggered by {event.key}
      </h3>
      <EventDataSection event={event} />
    </>
  );
}

function EventDataSection({ event }: { event: Event }) {
  const getEventDataQuery = useQuery({
    ...queries.events.getData(event.metadata.id),
  });

  if (getEventDataQuery.isLoading || !getEventDataQuery.data) {
    return (
      <div className="flex flex-row flex-1 w-full h-full">
        <Icons.spinner className="mr-2 h-4 w-4 animate-spin" />
      </div>
    );
  }

  const eventData = getEventDataQuery.data;

  return (
    <>
      <Code language="json" className="my-4" maxHeight="400px">
        {JSON.stringify(JSON.parse(eventData.data), null, 2)}
      </Code>
    </>
  );
}

function TriggeringCronSection({ cron }: { cron: string }) {
  const prettyInterval = `Runs ${CronPrettifier.toString(
    cron,
  ).toLowerCase()} UTC`;

  return (
    <>
      <h3 className="text-xl font-semibold leading-tight text-foreground mb-4">
        Triggered by Cron
      </h3>
      <div className="text-sm text-muted-foreground">{prettyInterval}</div>
      <Code language="typescript" className="my-4">
        {cron}
      </Code>
    </>
  );
}
