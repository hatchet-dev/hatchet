import React, { useMemo, useState, useRef } from 'react';
import { Step, StepRun, StepRunStatus } from '@/lib/api';
import { cn, formatDuration } from '@/lib/utils';
import { IconType } from 'react-icons';
import {
  BiHourglass,
  BiPlay,
  BiCheck,
  BiX,
  BiBlock,
  BiPause,
} from 'react-icons/bi';
import { TabOption } from './step-run-detail/step-run-detail';

const statusAnimations: Record<StepRunStatus, string> = {
  [StepRunStatus.PENDING]: 'animate-pulse',
  [StepRunStatus.PENDING_ASSIGNMENT]: 'animate-pulse',
  [StepRunStatus.ASSIGNED]: 'animate-bounce',
  [StepRunStatus.RUNNING]: 'animate-spin',
  [StepRunStatus.SUCCEEDED]: '',
  [StepRunStatus.FAILED]: '',
  [StepRunStatus.CANCELLED]: '',
  [StepRunStatus.CANCELLING]: 'animate-pulse',
};

const statusIcons: Record<StepRunStatus, IconType> = {
  [StepRunStatus.PENDING]: BiHourglass,
  [StepRunStatus.PENDING_ASSIGNMENT]: BiHourglass,
  [StepRunStatus.ASSIGNED]: BiPlay,
  [StepRunStatus.RUNNING]: BiHourglass, // Using hourglass for running as there's no direct spinning icon
  [StepRunStatus.SUCCEEDED]: BiCheck,
  [StepRunStatus.FAILED]: BiX,
  [StepRunStatus.CANCELLED]: BiBlock,
  [StepRunStatus.CANCELLING]: BiPause,
};

interface MiniMapProps {
  steps?: Step[];
  stepRuns?: StepRun[];
  selectedStepRunId?: string;
  onClick: (stepRunId?: string, defaultOpenTab?: TabOption) => void;
}

export const MiniMap: React.FC<MiniMapProps> = ({
  steps = [],
  stepRuns = [],
  selectedStepRunId,
  onClick,
}) => {
  const [hoveredStepId, setHoveredStepId] = useState<string | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);

  const columns = useMemo(() => {
    const columns: Step[][] = [];
    const processed = new Set<string>();

    const addToColumn = (step: Step, columnIndex: number) => {
      if (!columns[columnIndex]) {
        columns[columnIndex] = [];
      }
      columns[columnIndex].push(step);
      processed.add(step.metadata.id);
    };

    const processStep = (step: Step) => {
      if (processed.has(step.metadata.id)) {
        return;
      }

      if (!step.parents || step.parents.length === 0) {
        addToColumn(step, 0);
      } else {
        const maxParentColumn = Math.max(
          ...step.parents.map((parentId) => {
            const parentStep = steps.find((s) => s.metadata.id === parentId);
            return parentStep
              ? columns.findIndex((col) => col.includes(parentStep))
              : -1;
          }),
        );

        addToColumn(step, maxParentColumn + 1);
      }
    };

    while (processed.size < steps.length) {
      steps.forEach(processStep);
    }

    return columns;
  }, [steps]);

  const isParentOfHovered = (step: Step) => {
    if (!hoveredStepId) {
      return false;
    }
    const hoveredStep = steps.find((s) => s.metadata.id === hoveredStepId);
    return hoveredStep?.parents?.includes(step.metadata.id) || false;
  };

  const normalizedStepRunsByStepId = useMemo(() => {
    return steps.reduce(
      (acc, step) => {
        const stepRun = stepRuns?.find((sr) => sr.stepId === step.metadata.id);
        if (stepRun) {
          acc[step.metadata.id] = stepRun;
        }
        return acc;
      },
      {} as Record<string, StepRun>,
    );
  }, [steps, stepRuns]);

  return (
    <div
      ref={containerRef}
      className="flex flex-row overflow-x-auto p-4 rounded-sm relative gap-1 border-gray-800 bg-slate-100 dark:bg-slate-900"
    >
      {columns.map((column, colIndex) => (
        <div
          key={colIndex}
          className="flex flex-col items-center justify-start h-full"
          style={{
            width: `${100 / columns.length}%`,
            minWidth: '100px', // Minimum width for readability
          }}
        >
          {column.map((step) => {
            const stepRun = normalizedStepRunsByStepId[step.metadata.id];
            const status = stepRun?.status || StepRunStatus.PENDING;
            const StatusIcon = statusIcons[status];

            const res: JSX.Element[] = [
              <div
                key={step.metadata.id}
                data-step-id={step.metadata.id}
                className={cn(
                  `shadow-md rounded-sm py-3 px-1 mb-1 w-full text-xs text-[#050c1c] dark:text-[#ffffff] font-semibold font-mono`,
                  `transition-all duration-300 ease-in-out`,
                  `cursor-pointer`,
                  `flex flex-row items-center justify-between border-2 dark:border-[1px]`,
                  `bg-[#ffffff] dark:bg-[#050c1c]`,
                  isParentOfHovered(step) ||
                    hoveredStepId === step.metadata.id ||
                    stepRun?.metadata.id === selectedStepRunId
                    ? 'border-indigo-500 dark:border-indigo-500'
                    : 'border-[#050c1c] dark:border-gray-400',
                )}
                style={{
                  height: '20px', // Fixed height for each step
                  opacity:
                    !selectedStepRunId ||
                    stepRun?.metadata.id === selectedStepRunId
                      ? 1
                      : 0.4,
                }}
                onMouseEnter={() => setHoveredStepId(step.metadata.id)}
                onMouseLeave={() => setHoveredStepId(null)}
                onClick={() => onClick(stepRun?.metadata.id)}
              >
                <div className="flex flex-row items-center justify-start">
                  <StatusIcon
                    className={cn('mr-1', statusAnimations[status])}
                  />
                  <div className="truncate flex-grow">{step.readableId}</div>
                </div>
                {stepRun.finishedAtEpoch && stepRun.startedAtEpoch && (
                  <div className="text-xs text-gray-500 dark:text-gray-400">
                    {formatDuration(
                      stepRun.finishedAtEpoch - stepRun.startedAtEpoch,
                    )}
                  </div>
                )}
              </div>,
            ];

            if (stepRun?.childWorkflowsCount) {
              res.push(
                <div
                  key={`${step.metadata.id}-child-workflows`}
                  className={cn(
                    `w-[calc(100%-1rem)] box-border shadow-md ml-4 rounded-sm py-3 px-2 mb-1 text-xs text-[#050c1c] dark:text-[#ffffff] font-semibold font-mono`,
                    `transition-all duration-300 ease-in-out`,
                    `cursor-pointer`,
                    `flex flex-row items-center justify-start border-2 dark:border-[1px]`,
                    `bg-[#ffffff] dark:bg-[#050c1c]`,
                    isParentOfHovered(step) ||
                      hoveredStepId === step.metadata.id ||
                      stepRun?.metadata.id === selectedStepRunId
                      ? 'border-indigo-500 dark:border-indigo-500'
                      : 'border-[#050c1c] dark:border-gray-400',
                  )}
                  style={{
                    height: '20px', // Fixed height for each step
                    opacity:
                      !selectedStepRunId ||
                      stepRun?.metadata.id === selectedStepRunId
                        ? 1
                        : 0.4,
                  }}
                  onMouseEnter={() => setHoveredStepId(step.metadata.id)}
                  onMouseLeave={() => setHoveredStepId(null)}
                  onClick={() =>
                    onClick(stepRun?.metadata.id, TabOption.ChildWorkflowRuns)
                  }
                >
                  <div className="truncate flex-grow">
                    {step.readableId}: {stepRun.childWorkflowsCount} children
                  </div>
                </div>,
              );
            }

            return res;
          })}
        </div>
      ))}
    </div>
  );
};
