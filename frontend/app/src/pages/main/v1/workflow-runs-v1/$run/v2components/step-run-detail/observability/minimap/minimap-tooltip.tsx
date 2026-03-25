import { formatDuration } from "../utils/format-utils";
import { getSpanColor } from "../utils/span-tree-utils";
import type { SpanMarker } from "./minimap-types";
import { cn } from "@/lib/utils";
import { createPortal } from "react-dom";

interface MinimapTooltipProps {
  marker: SpanMarker;
  position: { x: number; y: number };
}

export function MinimapTooltip({ marker, position }: MinimapTooltipProps) {
  return createPortal(
    <div
      className="z-50 overflow-hidden rounded-lg border border-border bg-popover py-1 shadow-lg"
      style={{
        position: "fixed",
        left: Math.min(position.x + 12, window.innerWidth - 220),
        top: position.y + 16,
        minWidth: 180,
        pointerEvents: "none",
      }}
    >
      <div className="truncate px-3 py-1.5 font-mono text-xs text-foreground">
        {marker.spanName}
      </div>
      <div className="flex items-center gap-2 px-3 py-1.5">
        <span
          className={cn(
            "size-2 shrink-0 rounded-full",
            getSpanColor(marker.span),
          )}
        />
        <span className="flex-1 font-mono text-xs text-muted-foreground">
          {marker.inProgress
            ? "In Progress"
            : marker.hasErrorInTree
              ? "Error"
              : "OK"}
        </span>
        <span className="font-mono text-xs text-foreground">
          {formatDuration(marker.durationMs)}
        </span>
      </div>
    </div>,
    document.body,
  );
}
