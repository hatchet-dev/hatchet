import type { DragMode, TimeRange } from "./minimap-types";
import { cn } from "@/lib/utils";

interface RangeHandlesProps {
  visibleRange: TimeRange;
  dragging: DragMode;
  startDrag: (
    mode: DragMode,
    e: React.PointerEvent,
    anchorPct?: number,
  ) => void;
}

export function RangeHandles({
  visibleRange,
  dragging,
  startDrag,
}: RangeHandlesProps) {
  const sPct = visibleRange.startPct * 100;
  const ePct = visibleRange.endPct * 100;

  return (
    <>
      {dragging !== "brush" && (
        <div
          className="pointer-events-none absolute inset-y-0 left-0 z-[1] bg-background/70"
          style={{ width: `${sPct}%` }}
        />
      )}

      {dragging !== "brush" && (
        <div
          className="pointer-events-none absolute inset-y-0 right-0 z-[1] bg-background/70"
          style={{ width: `${100 - ePct}%` }}
        />
      )}

      <div
        className={cn(
          "pointer-events-none absolute inset-y-0 z-[3] transition-opacity duration-150",
          dragging === "brush"
            ? "opacity-0"
            : dragging
              ? "opacity-100"
              : "opacity-0 group-hover:opacity-100",
        )}
        style={{ left: `${sPct}%`, right: `${100 - ePct}%` }}
      >
        <div
          className="pointer-events-auto absolute inset-y-[3px] left-0 flex w-[20px] items-center justify-center rounded-full border border-border/40 bg-muted"
          style={{ cursor: "ew-resize" }}
          onPointerDown={(e) => startDrag("left", e)}
        >
          <svg
            width="6"
            height="10"
            viewBox="0 0 6 10"
            fill="none"
            className="text-muted-foreground/80"
          >
            <path
              d="M4 1L1 5L4 9"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>

        <div
          className="pointer-events-auto absolute inset-y-[3px] right-0 flex w-[20px] items-center justify-center rounded-full border border-border/40 bg-muted"
          style={{ cursor: "ew-resize" }}
          onPointerDown={(e) => startDrag("right", e)}
        >
          <svg
            width="6"
            height="10"
            viewBox="0 0 6 10"
            fill="none"
            className="text-muted-foreground/80"
          >
            <path
              d="M2 1L5 5L2 9"
              stroke="currentColor"
              strokeWidth="1.5"
              strokeLinecap="round"
              strokeLinejoin="round"
            />
          </svg>
        </div>
      </div>
    </>
  );
}
