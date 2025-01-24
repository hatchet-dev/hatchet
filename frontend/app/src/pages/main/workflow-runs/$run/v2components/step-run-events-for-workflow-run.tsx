import {
  StepRun,
  queries,
  WorkflowRunShape,
  Step,
  WorkflowRunStatus,
} from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';
import { DataTable } from '@/components/molecules/data-table/data-table';
import { ActivityEventData, columns } from './events-columns';
import { useTenant } from '@/lib/atoms';
import { useLocation } from 'react-router-dom';

export function StepRunEvents({
  workflowRun,
  filteredStepRunId,
  onClick,
}: {
  workflowRun: WorkflowRunShape;
  filteredStepRunId?: string;
  onClick?: (stepRunId?: string) => void;
}) {
  const tenant = useTenant();
  const tenantId = tenant.tenant?.metadata.id;
  const location = useLocation();
  const taskRunId = location.pathname.split('/')[-1];

  const eventsQuery = useQuery({
    ...queries.v2StepRunEvents.list(tenantId, taskRunId, {
      limit: 50,
      offset: 0,
    }),
    refetchInterval: () => {
      // if (workflowRun.status === WorkflowRunStatus.RUNNING) {
      //   return 1000;
      // }

      return 5000;
    },
  });

  const events = eventsQuery.data?.rows || [];

  const stepRuns = useMemo(() => {
    return (
      workflowRun.jobRuns?.flatMap((jr) => jr.stepRuns).filter((x) => !!x) ||
      ([] as StepRun[])
    );
  }, [workflowRun]);

  const steps = useMemo(() => {
    return (
      (
        workflowRun.jobRuns
          ?.flatMap((jr) => jr.job?.steps)
          .filter((x) => !!x) || ([] as Step[])
      ).flatMap((x) => x) || ([] as Step[])
    );
  }, [workflowRun]);

  const normalizedStepRunsByStepRunId = useMemo(() => {
    return stepRuns.reduce(
      (acc, stepRun) => {
        if (!stepRun) {
          return acc;
        }

        acc[stepRun.metadata.id] = stepRun;
        return acc;
      },
      {} as Record<string, StepRun>,
    );
  }, [stepRuns]);

  const normalizedStepsByStepRunId = useMemo(() => {
    return stepRuns.reduce(
      (acc, stepRun) => {
        if (!stepRun) {
          return acc;
        }

        const step = steps?.find((s) => s?.metadata.id === stepRun.stepId);
        if (step && stepRun) {
          acc[stepRun.metadata.id] = step;
        }
        return acc;
      },
      {} as Record<string, Step>,
    );
  }, [steps, stepRuns]);

  const tableData: ActivityEventData[] =
    events?.map((item) => {
      return {
        metadata: {
          id: '' + item.id,
          createdAt: item.timestamp,
          updatedAt: item.timestamp,
        },
        event: item,
        stepRun: item.taskId
          ? normalizedStepRunsByStepRunId[item.taskId]
          : undefined,
        step: item.taskId ? normalizedStepsByStepRunId[item.taskId] : undefined,
      };
    }) || [];

  const cols = columns({
    onRowClick: onClick
      ? (row) => onClick(row.stepRun?.metadata.id)
      : undefined,
    allEvents: tableData,
  });

  return (
    <DataTable
      emptyState={<>No events found.</>}
      isLoading={eventsQuery.isLoading}
      columns={cols}
      filters={[]}
      data={tableData}
    />
  );
}
