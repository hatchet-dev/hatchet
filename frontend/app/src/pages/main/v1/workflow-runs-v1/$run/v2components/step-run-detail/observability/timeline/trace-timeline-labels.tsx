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
import { Button } from '@/components/v1/ui/button';
import { cn } from '@/lib/utils';
import { ChevronRight, ChevronDown, AlertCircle } from 'lucide-react';
import { memo, type ReactNode } from 'react';

function ConnectorLines({
  depth,
  connectorFlags,
  isLastChild,
}: {
  depth: number;
  connectorFlags: boolean[];
  isLastChild?: boolean;
}) {
  return (
    <>
      {Array.from({ length: depth }).map((_, i) => {
        const isOwnLevel = i === depth - 1;
        const showLine =
          isLastChild !== undefined && isOwnLevel
            ? connectorFlags[i] || !isLastChild
            : connectorFlags[i];
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
    </>
  );
}

function ExpandToggle({
  isExpanded,
  hasChildren = true,
  onToggle,
}: {
  isExpanded: boolean;
  hasChildren?: boolean;
  onToggle: () => void;
}) {
  if (!hasChildren) {
    return (
      <div
        style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
        className="shrink-0"
      />
    );
  }
  return (
    <button
      className="flex shrink-0 items-center justify-center text-muted-foreground transition-colors hover:text-foreground"
      style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
      onClick={(e) => {
        e.stopPropagation();
        onToggle();
      }}
    >
      {isExpanded ? (
        <ChevronDown className="size-3" />
      ) : (
        <ChevronRight className="size-3" />
      )}
    </button>
  );
}

function LabelRow({
  selected,
  dimmed,
  onClick,
  children,
}: {
  selected?: boolean;
  dimmed?: boolean;
  onClick?: () => void;
  children: ReactNode;
}) {
  return (
    <div
      className={cn(
        'flex shrink-0 items-center px-2',
        onClick && 'cursor-pointer rounded-l transition-colors',
        selected ? 'bg-primary/8' : onClick && 'hover:bg-muted/50',
        dimmed && 'opacity-40',
      )}
      style={{ height: ROW_HEIGHT }}
      onClick={onClick}
    >
      {children}
    </div>
  );
}

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
            <LabelRow key={row.rowKey}>
              <ConnectorLines
                depth={row.depth}
                connectorFlags={row.connectorFlags}
              />
              <div
                style={{ width: CONNECTOR_WIDTH + CONNECTOR_GAP }}
                className="shrink-0"
              />
              <Button
                variant="link"
                size="xs"
                className="truncate text-sm"
                onClick={() =>
                  onShowMore(row.groupId, row.currentVisible + SHOW_MORE_BATCH)
                }
              >
                Show {Math.min(row.remaining, SHOW_MORE_BATCH)} more
                {row.remaining > SHOW_MORE_BATCH &&
                  ` (${row.remaining.toLocaleString()} remaining)`}
              </Button>
            </LabelRow>
          );
        }

        if (row.kind === 'group') {
          const isSelected = selectedGroupId === row.group.groupId;
          return (
            <LabelRow
              key={row.rowKey}
              selected={isSelected}
              onClick={() => {
                expandOnly(row.group.groupId);
                onGroupSelect?.(row.group);
              }}
            >
              <ConnectorLines
                depth={row.depth}
                connectorFlags={row.connectorFlags}
                isLastChild={row.isLastChild}
              />
              <ExpandToggle
                isExpanded={row.isExpanded}
                onToggle={() => toggleExpand(row.group.groupId)}
              />
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
            </LabelRow>
          );
        }

        const isSelected = selectedSpan?.spanId === row.span.spanId;
        const displayName = getDisplayName(row.span);

        return (
          <LabelRow
            key={row.rowKey}
            selected={isSelected}
            dimmed={!row.matchesFilter}
            onClick={() => {
              if (row.hasChildren) {
                expandOnly(row.rowKey);
              }
              onSpanSelect?.(row.span);
            }}
          >
            <ConnectorLines
              depth={row.depth}
              connectorFlags={row.connectorFlags}
              isLastChild={row.isLastChild}
            />
            <ExpandToggle
              isExpanded={row.isExpanded}
              hasChildren={row.hasChildren}
              onToggle={() => toggleExpand(row.rowKey)}
            />
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
          </LabelRow>
        );
      })}
    </>
  );
});
