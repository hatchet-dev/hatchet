import { useMemo } from 'react';
import { queries, V1TaskSummary } from '@/lib/api';
import { TabOption } from './step-run-detail/step-run-detail';
import StepRunNode from './step-run-node';
import { useWorkflowDetails } from '../../hooks/workflow-details';
import { useQuery } from '@tanstack/react-query';

interface JobMiniMapProps {
  onClick: (stepRunId?: string, defaultOpenTab?: TabOption) => void;
}

type NodeRelationship = {
  node: string;
  children: string[];
  parents: string[];
};

export const JobMiniMap = ({ onClick }: JobMiniMapProps) => {
  const { shape, taskRuns: tasks, isLoading, isError } = useWorkflowDetails();

  const taskRunRelationships: NodeRelationship[] = useMemo(
    () =>
      tasks.map((item) => {
        const node = item.taskExternalId;
        const children =
          shape.find((i) => i.taskExternalId === node)?.childrenExternalIds ||
          [];
        const parents = shape
          .filter((i) => i.childrenExternalIds.includes(node))
          .map((i) => i.taskExternalId);

        return {
          node,
          children,
          parents,
        };
      }) || [],
    [shape, tasks],
  );

  const columns = useMemo(() => {
    const columns: V1TaskSummary[][] = [];
    const processed = new Set<string>();

    const addToColumn = (taskRun: V1TaskSummary, columnIndex: number) => {
      if (!columns[columnIndex]) {
        columns[columnIndex] = [];
      }

      columns[columnIndex].push(taskRun);
      processed.add(taskRun.taskExternalId);
    };

    const processTaskRun = (taskRun: V1TaskSummary) => {
      if (processed.has(taskRun.taskExternalId)) {
        return;
      }

      const relationship = taskRunRelationships.find(
        (r) => r.node == taskRun.taskExternalId,
      );

      if (!relationship || relationship.parents.length === 0) {
        addToColumn(taskRun, 0);
      } else {
        const maxParentColumn = Math.max(
          ...relationship.parents.map((parentId) => {
            const parentStep = tasks.find((r) => r.taskExternalId === parentId);

            return parentStep
              ? columns.findIndex((col) => col.includes(parentStep))
              : -1;
          }),
        );

        if (maxParentColumn > -1) {
          addToColumn(taskRun, maxParentColumn + 1);
        }
      }
    };

    while (processed.size < tasks.length) {
      tasks.forEach(processTaskRun);
    }

    return columns;
  }, [taskRunRelationships, tasks]);

  if (isLoading || isError) {
    return null;
  }

  return (
    <div className="flex flex-1 flex-row p-4 rounded-sm relative gap-1">
      {columns.map((column, colIndex) => (
        <div
          key={colIndex}
          className="flex flex-col justify-start h-full min-w-fit grow"
        >
          {column.map((taskRun) => {
            return (
              <StepRunNode
                key={taskRun.taskExternalId}
                data={{
                  task: taskRun,
                  graphVariant: 'none',
                  onClick: () => onClick(taskRun.metadata.id),
                  childWorkflowsCount: 0,
                  taskName:
                    shape.find((i) => i.taskExternalId === taskRun.metadata.id)
                      ?.taskName || '',
                }}
              />
            );
          })}
        </div>
      ))}
    </div>
  );
};

type UseTaskRunProps = {
  taskRunId: string;
};

export const useTaskRun = ({ taskRunId }: UseTaskRunProps) => {
  const taskRunQuery = useQuery({
    ...queries.v1Tasks.get(taskRunId),
    refetchInterval: 5000,
  });

  return {
    taskRun: taskRunQuery.data,
    isLoading: taskRunQuery.isLoading,
    isError: taskRunQuery.isError,
  };
};

export const TaskRunMiniMap = ({
  onClick,
  taskRunId,
}: JobMiniMapProps & UseTaskRunProps) => {
  const { taskRun, isLoading, isError } = useTaskRun({ taskRunId });

  if (isLoading || isError || !taskRun) {
    return null;
  }

  return (
    <div className="flex flex-1 flex-row p-4 rounded-sm relative gap-1">
      <div className="flex flex-col justify-start w-full h-fit grow">
        <StepRunNode
          key={taskRun.taskExternalId}
          data={{
            task: taskRun,
            graphVariant: 'none',
            onClick: () => onClick(taskRun.metadata.id),
            childWorkflowsCount: 0,
            taskName: taskRun.displayName,
          }}
        />
      </div>
    </div>
  );
};
