import {
  ROW_HEIGHT,
  type FlatGroupRow,
  type SpanGroupInfo,
} from './trace-timeline-utils';
import { cn } from '@/lib/utils';
import { memo, type MouseEvent } from 'react';

interface GroupBarProps {
  row: FlatGroupRow;
  timelineMaxMs: number;
  visMinStart: number;
  hasAnyInProgress: boolean;
  hasAnyLiveQueued: boolean;
  isSelected: boolean;
  isHovered: boolean;
  onHover: (rowKey: string | null, event?: MouseEvent) => void;
  onMouseMove: (e: MouseEvent) => void;
  onGroupSelect?: (group: SpanGroupInfo) => void;
  expandOnly: (id: string) => void;
}

export const GroupBar = memo(function GroupBar({
  row,
  timelineMaxMs,
  visMinStart,
  hasAnyInProgress,
  hasAnyLiveQueued,
  isSelected,
  isHovered,
  onHover,
  onMouseMove,
  onGroupSelect,
  expandOnly,
}: GroupBarProps) {
  const durationMs = row.group.latestEndMs - row.group.earliestStartMs;
  const leftPct =
    timelineMaxMs > 0
      ? ((row.group.earliestStartMs - visMinStart) / timelineMaxMs) * 100
      : 0;
  const widthPct = timelineMaxMs > 0 ? (durationMs / timelineMaxMs) * 100 : 0;
  const hasErrors = row.group.errorCount > 0;
  const noTransition = hasAnyInProgress || hasAnyLiveQueued;

  return (
    <div
      className={cn(
        'relative shrink-0 transition-colors',
        isSelected && 'bg-primary/8',
      )}
      style={{ height: ROW_HEIGHT }}
    >
      <div
        className={cn(
          'absolute bottom-[10px] top-[10px] cursor-pointer rounded-[2px]',
          !noTransition && 'transition-all',
          hasErrors ? 'bg-red-500' : 'bg-green-500',
          isSelected
            ? 'ring-2 ring-primary ring-offset-1 ring-offset-background'
            : isHovered
              ? 'ring-1 ring-foreground/20'
              : '',
        )}
        style={{
          left: `${leftPct}%`,
          width: `${Math.max(widthPct, 0.3)}%`,
          minWidth: 2,
        }}
        onMouseEnter={(e) => onHover(row.rowKey, e)}
        onMouseMove={onMouseMove}
        onMouseLeave={() => onHover(null)}
        onClick={() => {
          expandOnly(row.group.groupId);
          onGroupSelect?.(row.group);
        }}
      />
    </div>
  );
});
