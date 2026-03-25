import { formatDuration } from '../utils/format-utils';
import type { BrushRange } from './use-brush-zoom';
import { memo } from 'react';

interface TimelineTickHeaderProps {
  ticks: number[];
  timelineMaxMs: number;
  visOffsetMs: number;
  effectiveCursorPct: number | null;
  brushRange: BrushRange | null;
}

export const TimelineTickHeader = memo(function TimelineTickHeader({
  ticks,
  timelineMaxMs,
  visOffsetMs,
  effectiveCursorPct,
  brushRange,
}: TimelineTickHeaderProps) {
  return (
    <div className="relative h-6 shrink-0">
      {ticks.map((t, i) => {
        const isLast = i === ticks.length - 1;
        return (
          <div
            key={t}
            className="absolute flex h-full items-center"
            style={{
              left: `${(t / timelineMaxMs) * 100}%`,
              transform: isLast ? 'translateX(-100%)' : undefined,
            }}
          >
            <span className="whitespace-nowrap font-mono text-xs uppercase tracking-wider text-muted-foreground">
              {formatDuration(t + visOffsetMs)}
            </span>
          </div>
        );
      })}

      {effectiveCursorPct !== null && !brushRange && (
        <div
          className="pointer-events-none absolute z-10 flex h-full items-center whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background"
          style={{
            left: `${effectiveCursorPct * 100}%`,
            transform:
              effectiveCursorPct < 0.05
                ? 'none'
                : effectiveCursorPct > 0.95
                  ? 'translateX(-100%)'
                  : 'translateX(-50%)',
          }}
        >
          {formatDuration(
            timelineMaxMs * effectiveCursorPct + visOffsetMs,
          )}
        </div>
      )}

      {brushRange && (
        <div
          className="pointer-events-none absolute z-20 flex h-full items-center"
          style={{
            left: `${brushRange.lo * 100}%`,
            width: `${(brushRange.hi - brushRange.lo) * 100}%`,
          }}
        >
          <span className="shrink-0 whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background">
            {formatDuration(timelineMaxMs * brushRange.lo + visOffsetMs)}
          </span>
          <div className="flex min-w-1 flex-1 items-center">
            <svg
              width="5"
              height="6"
              viewBox="0 0 5 6"
              className="shrink-0 fill-primary"
            >
              <path d="M5 0L0 3L5 6Z" />
            </svg>
            <div className="h-px flex-1 bg-primary" />
          </div>
          <span className="shrink-0 whitespace-nowrap rounded bg-primary px-1.5 py-0.5 font-mono text-[10px] font-medium leading-tight text-primary-foreground">
            {formatDuration(
              timelineMaxMs * (brushRange.hi - brushRange.lo),
            )}
          </span>
          <div className="flex min-w-1 flex-1 items-center">
            <div className="h-px flex-1 bg-primary" />
            <svg
              width="5"
              height="6"
              viewBox="0 0 5 6"
              className="shrink-0 fill-primary"
            >
              <path d="M0 0L5 3L0 6Z" />
            </svg>
          </div>
          <span className="shrink-0 whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background">
            {formatDuration(timelineMaxMs * brushRange.hi + visOffsetMs)}
          </span>
        </div>
      )}
    </div>
  );
});
