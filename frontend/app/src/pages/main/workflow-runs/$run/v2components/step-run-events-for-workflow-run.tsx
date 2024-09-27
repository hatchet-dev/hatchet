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

export function StepRunEvents({
  workflowRun,
  filteredStepRunId,
  onClick,
}: {
  workflowRun: WorkflowRunShape;
  filteredStepRunId?: string;
  onClick?: (stepRunId?: string) => void;
}) {
  const eventsQuery = useQuery({
    ...queries.workflowRuns.listStepRunEvents(
      workflowRun.tenantId,
      workflowRun.metadata.id,
    ),
    refetchInterval: () => {
      if (workflowRun.status === WorkflowRunStatus.RUNNING) {
        return 1000;
      }

      return 5000;
    },
  });

  const filteredEvents = useMemo(() => {
    if (!filteredStepRunId) {
      return eventsQuery.data?.rows || [];
    }

    return eventsQuery.data?.rows?.filter(
      (x) => x.stepRunId === filteredStepRunId,
    );
  }, [eventsQuery.data, filteredStepRunId]);

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
    filteredEvents?.map((item) => {
      return {
        metadata: {
          id: '' + item.id,
          createdAt: item.timeFirstSeen,
          updatedAt: item.timeLastSeen,
        },
        event: item,
        stepRun: item.stepRunId
          ? normalizedStepRunsByStepRunId[item.stepRunId]
          : undefined,
        step: item.stepRunId
          ? normalizedStepsByStepRunId[item.stepRunId]
          : undefined,
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
