import React, { useMemo, useState, useRef, useEffect } from 'react';
import { Step, StepRun, StepRunStatus } from '@/lib/api';
import { cn } from '@/lib/utils';
import { IconType } from 'react-icons';
import {
  BiHourglass,
  BiPlay,
  BiCheck,
  BiX,
  BiBlock,
  BiPause,
} from 'react-icons/bi';

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

const statusColors: Record<StepRunStatus, string> = {
  [StepRunStatus.PENDING]: 'bg-gray-300',
  [StepRunStatus.PENDING_ASSIGNMENT]: 'bg-yellow-300',
  [StepRunStatus.ASSIGNED]: 'bg-blue-300',
  [StepRunStatus.RUNNING]: 'bg-blue-500',
  [StepRunStatus.SUCCEEDED]: 'bg-green-500',
  [StepRunStatus.FAILED]: 'bg-red-500',
  [StepRunStatus.CANCELLED]: 'bg-gray-500',
  [StepRunStatus.CANCELLING]: 'bg-orange-500',
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
  onClick: (stepRunId?: string) => void;
}

export const MiniMap: React.FC<MiniMapProps> = ({
  steps = [],
  stepRuns = [],
  selectedStepRunId,
  onClick,
}) => {
  const [hoveredStepId, setHoveredStepId] = useState<string | null>(null);
  const [lines, setLines] = useState<
    { x1: number; y1: number; x2: number; y2: number }[]
  >([]);
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

  useEffect(() => {
    if (hoveredStepId && containerRef.current) {
      const hoveredElement = containerRef.current.querySelector(
        `[data-step-id="${hoveredStepId}"]`,
      );
      const hoveredStep = steps.find((s) => s.metadata.id === hoveredStepId);

      if (hoveredElement && hoveredStep?.parents) {
        const parentElements = hoveredStep.parents
          .map((parentId) =>
            containerRef.current?.querySelector(`[data-step-id="${parentId}"]`),
          )
          .filter(Boolean);

        const hoveredRect = hoveredElement.getBoundingClientRect();
        const containerRect = containerRef.current.getBoundingClientRect();

        const newLines = parentElements.map((parentElement) => {
          const parentRect = parentElement!.getBoundingClientRect();
          return {
            x1: parentRect.left + parentRect.width / 2 - containerRect.left,
            y1: parentRect.top + parentRect.height / 2 - containerRect.top,
            x2: hoveredRect.left + hoveredRect.width / 2 - containerRect.left,
            y2: hoveredRect.top + hoveredRect.height / 2 - containerRect.top,
          };
        });

        setLines(newLines);
      } else {
        setLines([]);
      }
    } else {
      setLines([]);
    }
  }, [hoveredStepId, steps]);

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
      className="flex flex-row overflow-x-auto p-4 bg-gray-100 rounded-lg relative gap-1"
    >
      <svg className="absolute top-0 left-0 w-full h-full pointer-events-none">
        {lines.map((line, index) => (
          <line
            key={index}
            x1={line.x1}
            y1={line.y1}
            x2={line.x2}
            y2={line.y2}
            stroke="rgba(59, 130, 246, 0.5)"
            strokeWidth="2"
          />
        ))}
      </svg>
      {columns.map((column, colIndex) => (
        <div
          key={colIndex}
          className="flex flex-col items-center justify-start h-full"
          style={{
            width: `${100 / columns.length}%`,
            minWidth: '100px', // Minimum width for readability
          }}
        >
          {column.map((step, stepIndex) => {
            const stepRun = normalizedStepRunsByStepId[step.metadata.id];
            const status = stepRun?.status || StepRunStatus.PENDING;
            const StatusIcon = statusIcons[status];
            return (
              <div
                key={step.metadata.id}
                data-step-id={step.metadata.id}
                className={cn(
                  `shadow-md rounded-lg p-1 mb-1 w-full text-xs text-gray-800`,
                  `transition-all duration-300 ease-in-out`,
                  `hover:shadow-lg hover:scale-105`,
                  `cursor-pointer`,
                  `flex flex-row items-center justify-start`,
                  statusColors[status],
                  isParentOfHovered(step) ? 'ring-2 ring-blue-500' : '',
                  hoveredStepId === step.metadata.id
                    ? 'ring-2 ring-blue-500'
                    : '',
                  stepRun?.metadata.id === selectedStepRunId
                    ? 'ring-2 ring-blue-500'
                    : '',
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
                <StatusIcon className={cn('mr-1', statusAnimations[status])} />
                <div className="truncate flex-grow">
                  {step.readableId} ({status})
                </div>
              </div>
            );
          })}
        </div>
      ))}
    </div>
  );
};
