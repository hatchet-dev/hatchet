import { useMemo } from 'react';
import {
  queries,
  V1TaskSummary,
  WorkflowRunShapeItemForWorkflowRunDetails,
} from '@/lib/api';
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

    while (processed.size < shape.length) {
      shape.forEach(processTaskRun);
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
