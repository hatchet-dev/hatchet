import { getDisplayName, isEngineSpan } from '../utils/span-tree-utils';
import {
  ROW_HEIGHT,
  CONNECTOR_WIDTH,
  CONNECTOR_GAP,
  SHOW_MORE_BATCH,
  type FlatRow,
  type SpanGroupInfo,
} from './trace-timeline-utils';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { cn } from '@/lib/utils';
import { ChevronRight, ChevronDown, AlertCircle } from 'lucide-react';
import { memo } from 'react';

interface TimelineLabelsProps {
  flatRows: FlatRow[];
  selectedSpan?: OtelSpanTree;
  selectedGroupId?: string;
  onSpanSelect?: (span: OtelSpanTree) => void;
  onGroupSelect?: (group: SpanGroupInfo) => void;
  onShowMore: (groupId: string, newVisibleCount: number) => void;
  toggleExpand: (id: string) => void;
  expandOnly: (id: string) => void;
}

export const TimelineLabels = memo(function TimelineLabels({
  flatRows,
  selectedSpan,
  selectedGroupId,
  onSpanSelect,
  onGroupSelect,
  onShowMore,
  toggleExpand,
  expandOnly,
}: TimelineLabelsProps) {
  return (
    <>
      {flatRows.map((row) => {
        if (row.kind === 'show-more') {
          return (
            <div
              key={row.rowKey}
              className="flex shrink-0 items-center px-2"
              style={{ height: ROW_HEIGHT }}
            >
              {Array.from({ length: row.depth }).map((_, i) => (
                <div
                  key={i}
                  className="flex shrink-0 items-center justify-center"
                  style={{ width: CONNECTOR_WIDTH, height: ROW_HEIGHT }}
                >
                  {row.connectorFlags[i] && (
                    <div className="h-full w-px bg-border" />
                  )}
                </div>
              ))}
              <div
                style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                className="shrink-0"
              />
              <button
                className="truncate text-sm text-primary hover:underline"
                onClick={() =>
                  onShowMore(row.groupId, row.currentVisible + SHOW_MORE_BATCH)
                }
              >
                Show {Math.min(row.remaining, SHOW_MORE_BATCH)} more
                {row.remaining > SHOW_MORE_BATCH &&
                  ` (${row.remaining.toLocaleString()} remaining)`}
              </button>
            </div>
          );
        }

        if (row.kind === 'group') {
          const isSelected = selectedGroupId === row.group.groupId;
          return (
            <div
              key={row.rowKey}
              className={cn(
                'flex shrink-0 cursor-pointer items-center rounded-l px-2 transition-colors',
                isSelected ? 'bg-primary/8' : 'hover:bg-muted/50',
              )}
              style={{ height: ROW_HEIGHT }}
              onClick={() => {
                expandOnly(row.group.groupId);
                onGroupSelect?.(row.group);
              }}
            >
              {Array.from({ length: row.depth }).map((_, i) => {
                const isOwnLevel = i === row.depth - 1;
                const showLine = isOwnLevel
                  ? row.connectorFlags[i] || !row.isLastChild
                  : row.connectorFlags[i];
                return (
                  <div
                    key={i}
                    className="flex shrink-0 items-center justify-center"
                    style={{ width: CONNECTOR_WIDTH, height: ROW_HEIGHT }}
                  >
                    {showLine && <div className="h-full w-px bg-border" />}
                  </div>
                );
              })}

              <button
                className="flex shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpand(row.group.groupId);
                }}
              >
                {row.isExpanded ? (
                  <ChevronDown className="size-3" />
                ) : (
                  <ChevronRight className="size-3" />
                )}
              </button>

              <span
                className={cn(
                  'truncate text-sm leading-tight',
                  isSelected
                    ? 'font-medium text-foreground'
                    : 'text-muted-foreground',
                )}
                title={`batch: ${row.group.groupName} (${row.group.totalCount.toLocaleString()} spans)`}
              >
                batch: {row.group.groupName}
              </span>
              <span className="ml-1.5 shrink-0 rounded bg-muted px-1.5 py-0.5 font-mono text-xs text-muted-foreground">
                {row.group.totalCount.toLocaleString()}
              </span>
              {row.group.errorCount > 0 && (
                <span className="ml-1 flex shrink-0 items-center gap-0.5 rounded bg-red-500/10 px-1.5 py-0.5 font-mono text-xs text-red-500">
                  <AlertCircle className="size-3" />
                  {row.group.errorCount.toLocaleString()}
                </span>
              )}
            </div>
          );
        }

        const isSelected = selectedSpan?.spanId === row.span.spanId;
        const isDimmed = !row.matchesFilter;
        const displayName = getDisplayName(row.span);

        return (
          <div
            key={row.rowKey}
            className={cn(
              'flex shrink-0 cursor-pointer items-center rounded-l px-2 transition-colors',
              isSelected ? 'bg-primary/8' : 'hover:bg-muted/50',
              isDimmed && 'opacity-40',
            )}
            style={{ height: ROW_HEIGHT }}
            onClick={() => {
              if (row.hasChildren) {
                expandOnly(row.rowKey);
              }
              onSpanSelect?.(row.span);
            }}
          >
            {Array.from({ length: row.depth }).map((_, i) => {
              const isOwnLevel = i === row.depth - 1;
              const showLine = isOwnLevel
                ? row.connectorFlags[i] || !row.isLastChild
                : row.connectorFlags[i];
              return (
                <div
                  key={i}
                  className="flex shrink-0 items-center justify-center"
                  style={{ width: CONNECTOR_WIDTH, height: ROW_HEIGHT }}
                >
                  {showLine && <div className="h-full w-px bg-border" />}
                </div>
              );
            })}

            {row.hasChildren ? (
              <button
                className="flex shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
                style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                onClick={(e) => {
                  e.stopPropagation();
                  toggleExpand(row.rowKey);
                }}
              >
                {row.isExpanded ? (
                  <ChevronDown className="size-3" />
                ) : (
                  <ChevronRight className="size-3" />
                )}
              </button>
            ) : (
              <div
                style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                className="shrink-0"
              />
            )}

            <span
              className={cn(
                'truncate text-sm leading-tight',
                isSelected
                  ? 'font-medium text-foreground'
                  : row.depth === 0
                    ? 'text-foreground'
                    : 'text-muted-foreground',
              )}
              title={displayName}
            >
              {displayName}
            </span>
            {isEngineSpan(row.span) && (
              <span className="ml-1.5 shrink-0 rounded bg-muted px-1 py-0.5 font-mono text-[10px] text-muted-foreground">
                engine
              </span>
            )}
          </div>
        );
      })}
    </>
  );
});
