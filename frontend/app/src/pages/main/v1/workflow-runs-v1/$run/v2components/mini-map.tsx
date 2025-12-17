import { useWorkflowDetails } from '../../hooks/use-workflow-details';
import { TabOption } from './step-run-detail/step-run-detail';
import StepRunNode from './step-run-node';
import { useRefetchInterval } from '@/contexts/refetch-interval-context';
import { queries, WorkflowRunShapeItemForWorkflowRunDetails } from '@/lib/api';
import { useQuery } from '@tanstack/react-query';
import { useMemo } from 'react';

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

  const taskRunRelationships: NodeRelationship[] = useMemo(() => {
    if (!shape || !tasks) {
      return [];
    }

    return shape.map((shapeItem) => {
      const node = shapeItem.stepId ?? 'placeholder';

      const children = shapeItem?.childrenStepIds || [];
      const parents = shape
        .filter((i) => node && i.childrenStepIds.includes(node))
        .map((i) => i.stepId);

      return {
        node,
        children,
        parents,
      };
    });
  }, [shape, tasks]);

  const columns = useMemo(() => {
    const columns: WorkflowRunShapeItemForWorkflowRunDetails[][] = [];
    const processed = new Set<string>();

    const addToColumn = (
      shapeItem: WorkflowRunShapeItemForWorkflowRunDetails,
      columnIndex: number,
    ) => {
      if (!columns[columnIndex]) {
        columns[columnIndex] = [];
      }

      columns[columnIndex].push(shapeItem);
      processed.add(shapeItem.stepId);
    };

    const processTaskRun = (
      shapeItem: WorkflowRunShapeItemForWorkflowRunDetails,
    ) => {
      if (processed.has(shapeItem.stepId)) {
        return;
      }

      const relationship = taskRunRelationships.find(
        (r) => r.node == shapeItem.stepId,
      );

      if (!relationship || relationship.parents.length === 0) {
        addToColumn(shapeItem, 0);
      } else {
        const maxParentColumn = Math.max(
          ...relationship.parents.map((parentId) => {
            const parentStep = shape.find((r) => r.stepId === parentId);

            return parentStep
              ? columns.findIndex((col) => col.includes(parentStep))
              : -1;
          }),
        );

        if (maxParentColumn > -1) {
          addToColumn(shapeItem, maxParentColumn + 1);
        }
      }
    };

    let iterations = 100;

    while (processed.size < shape.length) {
      if (iterations === 0) {
        break;
      }

      // IMPORTANT: This can cause an infinite loop when rendering if
      // nothing ever gets added to `processed`. Setting a max iterations to
      // hopefully prevent that
      shape.forEach(processTaskRun);
      iterations -= 1;
    }

    return columns;
  }, [taskRunRelationships, shape]);

  if (isLoading || isError) {
    return null;
  }

  return (
    <div className="relative flex flex-1 flex-row gap-1 rounded-sm p-4">
      {columns.map((column, colIndex) => (
        <div
          key={colIndex}
          className="flex h-full min-w-fit grow flex-col justify-start"
        >
          {column.map((shapeItem) => {
            const taskRun = tasks.find(
              (t) => t.metadata.id === shapeItem.taskExternalId,
            );
            return (
              <StepRunNode
                key={shapeItem.stepId}
                data={{
                  taskRun,
                  graphVariant: 'none',
                  onClick: () => onClick(taskRun?.metadata.id),
                  childWorkflowsCount: taskRun?.numSpawnedChildren || 0,
                  taskName: shapeItem.taskName,
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

const useTaskRun = ({ taskRunId }: UseTaskRunProps) => {
  const { refetchInterval } = useRefetchInterval();
  const taskRunQuery = useQuery({
    ...queries.v1Tasks.get(taskRunId),
    refetchInterval,
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
    <div className="relative flex flex-1 flex-row gap-1 rounded-sm p-4">
      <div className="flex h-fit w-full grow flex-col justify-start">
        <StepRunNode
          data={{
            taskRun,
            graphVariant: 'none',
            onClick: () => onClick(taskRun.metadata.id),
            childWorkflowsCount: taskRun.numSpawnedChildren,
            taskName: taskRun.displayName,
          }}
        />
      </div>
    </div>
  );
};
