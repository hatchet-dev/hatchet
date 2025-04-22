import { useRunDetail } from '@/next/hooks/use-run-detail';
import { useRuns, RunsProvider } from '@/next/hooks/use-runs';
import { V1TaskSummary, V1WorkflowRun } from '@/lib/api';
import { PropsWithChildren, useMemo, useState } from 'react';
import { Button } from '@/next/components/ui/button';
import { Timeline } from '../timeline';
import { TimelineProvider } from '@/next/hooks/use-timeline-context';
import { RunId } from './run-id';
import { ChevronRight } from 'lucide-react';
import { cn } from '@/next/lib/utils';
import { RunsBadge } from './runs-badge';
import { HiMiniArrowTurnLeftUp } from 'react-icons/hi2';
import { useNavigate } from 'react-router-dom';
import { ROUTES } from '@/next/lib/routes';

const MAX_CHILDREN = 10;
const MAX_DEPTH = 2;

interface RunRowProps {
  run?: V1TaskSummary;
  depth: number;
  isTitle?: boolean;
  hasChildren?: boolean;
  isExpanded?: boolean;
  toggleChildren?: () => void;
  parentRun?: V1WorkflowRun;
  onClick?: () => void;
}

function HighlightGroup({ children }: PropsWithChildren) {
  return <div className="">{children}</div>;
}

function RunRow({
  run,
  isTitle,
  depth,
  hasChildren,
  isExpanded,
  toggleChildren,
  parentRun,
  onClick,
}: RunRowProps) {
  return (
    <div
      className={cn(
        'grid grid-cols-[200px,1fr] items-center',
        isTitle && 'cursor-pointer',
      )}
      onClick={onClick}
    >
      <div
        className={cn(
          'text-sm text-muted-foreground truncate overflow-hidden whitespace-nowrap flex items-center gap-2',
          isTitle && 'cursor-pointer',
        )}
        style={{
          paddingLeft: `${depth * 15}px`,
        }}
      >
        {hasChildren && (
          <Button
            variant="ghost"
            size="icon"
            className="w-4 h-4"
            onClick={(e) => {
              e.stopPropagation();
              toggleChildren?.();
            }}
          >
            <ChevronRight
              className={cn('w-2 h-2', isExpanded ? 'rotate-90' : 'rotate-0')}
            />
          </Button>
        )}

        {run?.numSpawnedChildren}
        {isTitle ? (
          parentRun ? (
            <div className="flex items-center gap-2 cursor-pointer">
              <HiMiniArrowTurnLeftUp className="w-4 h-4" />
              <RunId wfRun={parentRun} />
            </div>
          ) : (
            <></>
          )
        ) : run ? (
          <>
            <RunId taskRun={run} />
            <RunsBadge status={run?.status} variant="xs" />
          </>
        ) : null}
      </div>
      <Timeline
        items={run ? [run] : []}
        showLabels={false}
        height={28}
        showTimeLabels={isTitle}
      />
    </div>
  );
}

interface RunRowWithChildrenProps {
  run: V1TaskSummary;
  depth: number;
  expandedIds: Set<string>;
  toggleExpanded: (id: string) => void;
  onTaskSelect?: (taskId: string) => void;
}

function RunRowWithChildren({
  run,
  depth,
  expandedIds,
  toggleExpanded,
  onTaskSelect,
}: RunRowWithChildrenProps) {
  const navigate = useNavigate();

  // TODO add parent...
  const { data: childrenData } = useRuns();

  const hasActualChildren = childrenData && childrenData.length > 0;
  const isExpanded = expandedIds.has(run.metadata.id);

  return (
    <HighlightGroup>
      <RunRow
        run={run}
        depth={depth}
        hasChildren={hasActualChildren}
        isExpanded={isExpanded}
        toggleChildren={() => toggleExpanded(run.metadata.id)}
        onClick={() => {
          if (depth == 0) {
            onTaskSelect?.(run.metadata.id);
          } else {
            navigate(ROUTES.runs.detail(run.metadata.id));
          }
        }}
      />
      {isExpanded && hasActualChildren && (
        <RunsProvider>
          <ChildrenList
            run={run}
            depth={depth}
            expandedIds={expandedIds}
            toggleExpanded={toggleExpanded}
            onTaskSelect={onTaskSelect}
          />
        </RunsProvider>
      )}
    </HighlightGroup>
  );
}

interface ChildrenListProps {
  run: V1TaskSummary;
  depth: number;
  expandedIds: Set<string>;
  toggleExpanded: (id: string) => void;
  onTaskSelect?: (taskId: string) => void;
}

function ChildrenList({
  run,
  depth,
  expandedIds,
  toggleExpanded,
  onTaskSelect,
}: ChildrenListProps) {
  // TODO add parent...
  const { data } = useRuns();
  const [maxChildren, setMaxChildren] = useState(MAX_CHILDREN);

  const [render, numHidden] = useMemo(() => {
    if (!data || data.length === 0) {
      return [[], 0];
    }

    const numHidden = data.length - maxChildren;
    return [data.slice(0, maxChildren), numHidden];
  }, [data, maxChildren]);

  if (depth > MAX_DEPTH) {
    return <>More...</>;
  }

  return (
    <div className="flex flex-col gap-0">
      {render
        ?.sort(
          (a, b) =>
            new Date(a.startedAt || 0).getTime() -
            new Date(b.startedAt || 0).getTime(),
        )
        .map((childRun) => (
          <RunRowWithChildren
            key={childRun.metadata.id}
            run={childRun}
            depth={depth + 1}
            expandedIds={expandedIds}
            toggleExpanded={toggleExpanded}
            onTaskSelect={onTaskSelect}
          />
        ))}
      {numHidden > 0 && (
        <div>
          <span>+{numHidden} more</span>
          <Button onClick={() => setMaxChildren(maxChildren + 10)}>
            Load more
          </Button>
        </div>
      )}
    </div>
  );
}

interface RunChildrenCardProps {
  workflow: V1WorkflowRun;
  parentRun?: V1WorkflowRun;
  onTaskSelect?: (taskId: string) => void;
}

export function RunChildrenCardRoot({
  workflow,
  parentRun,
  onTaskSelect,
}: RunChildrenCardProps) {
  const { data } = useRunDetail();
  const [expandedIds, setExpandedIds] = useState<Set<string>>(new Set());

  const run = data?.run;

  if (!run) {
    return <div>Run not found</div>;
  }

  const toggleExpanded = (id: string) => {
    setExpandedIds((prev) => {
      const newSet = new Set(prev);
      if (newSet.has(id)) {
        newSet.delete(id);
      } else {
        newSet.add(id);
      }
      return newSet;
    });
  };

  return (
    <TimelineProvider>
      <RunsProvider>
        <div className="flex flex-col gap-0">
          <HighlightGroup>
            <RunRow
              isTitle
              depth={0}
              parentRun={parentRun}
              onClick={() => onTaskSelect?.(workflow.metadata.id)}
            />
          </HighlightGroup>
          {data.tasks.map((task) => (
            <RunRowWithChildren
              key={task.metadata.id}
              run={task}
              depth={0}
              expandedIds={expandedIds}
              toggleExpanded={toggleExpanded}
              onTaskSelect={onTaskSelect}
            />
          ))}
        </div>
      </RunsProvider>
    </TimelineProvider>
  );
}
