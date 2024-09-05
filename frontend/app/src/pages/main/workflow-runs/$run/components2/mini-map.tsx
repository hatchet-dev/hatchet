import React, { useMemo, useState, useRef, useEffect } from 'react';
import { Step, StepRun } from '@/lib/api';

interface MiniMapProps {
  steps?: Step[];
  stepRuns?: StepRun[];
}

export const MiniMap: React.FC<MiniMapProps> = ({
  steps = [],
  stepRuns = [],
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
      className="flex overflow-x-auto p-4 bg-gray-100 rounded-lg relative"
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
          className="flex flex-col items-center mx-2 first:ml-0 last:mr-0"
        >
          {column.map((step) => (
            <div
              key={step.metadata.id}
              data-step-id={step.metadata.id}
              className={`
                bg-white shadow-md rounded-lg p-3 mb-3 w-32 text-center text-sm text-gray-800
                transition-all duration-300 ease-in-out
                ${isParentOfHovered(step) ? 'ring-2 ring-blue-500' : ''}
                ${hoveredStepId === step.metadata.id ? 'ring-2 ring-blue-500' : ''}
                hover:shadow-lg hover:scale-105
              `}
              onMouseEnter={() => setHoveredStepId(step.metadata.id)}
              onMouseLeave={() => setHoveredStepId(null)}
            >
              {step.readableId}
              {normalizedStepRunsByStepId[step.metadata.id] && (
                <div className="text-xs text-gray-500">
                  {normalizedStepRunsByStepId[step.metadata.id].status}
                </div>
              )}
            </div>
          ))}
        </div>
      ))}
    </div>
  );
};
