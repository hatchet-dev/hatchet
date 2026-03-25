import { formatDuration } from "../utils/format-utils";
import type { TimeRange } from "./minimap-types";

interface DragAnnotationProps {
  visibleRange: TimeRange;
  totalMs: number;
}

export function DragAnnotation({ visibleRange, totalMs }: DragAnnotationProps) {
  return (
    <>
      <div
        className="pointer-events-none absolute inset-y-0 z-[5] w-px bg-foreground/70"
        style={{ left: `${visibleRange.startPct * 100}%` }}
      />
      <div
        className="pointer-events-none absolute inset-y-0 z-[5] w-px bg-foreground/70"
        style={{ left: `${visibleRange.endPct * 100}%` }}
      />
      <div
        className="pointer-events-none absolute z-[8] flex items-center"
        style={{
          left: `${visibleRange.startPct * 100}%`,
          width: `${(visibleRange.endPct - visibleRange.startPct) * 100}%`,
          top: "50%",
          transform: "translateY(-50%)",
        }}
      >
        <span className="shrink-0 whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background">
          {formatDuration(totalMs * visibleRange.startPct)}
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
            totalMs * (visibleRange.endPct - visibleRange.startPct),
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
          {formatDuration(totalMs * visibleRange.endPct)}
        </span>
      </div>
    </>
  );
}
