import { CursorOverlay } from './cursor-overlay';
import { DragAnnotation } from './drag-annotation';
import { MinimapTooltip } from './minimap-tooltip';
import type { TraceMinimapProps } from './minimap-types';
import { collectSpanMarkers, pctFromEvent } from './minimap-utils';
import { RangeHandles } from './range-handles';
import { SpanMarkers } from './span-markers';
import { useMinimapDrag } from './use-minimap-drag';
import { useMemo, useRef, useState } from 'react';

export type { TimeRange } from './minimap-types';

export function TraceMinimap({
  spanTrees,
  minMs,
  maxMs,
  visibleRange,
  onRangeChange,
  expandedSpanIds,
  onSpanSelect,
  externalHoverPct,
  onHoverPctChange,
}: TraceMinimapProps) {
  const trackRef = useRef<HTMLDivElement>(null);

  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{
    x: number;
    y: number;
  } | null>(null);
  const [hoverPct, setHoverPct] = useState<number | null>(null);

  const totalMs = maxMs - minMs;
  const markers = useMemo(
    () => collectSpanMarkers(spanTrees, minMs, totalMs, expandedSpanIds),
    [spanTrees, minMs, totalMs, expandedSpanIds],
  );

  const { dragging, startDrag, handleTrackDown, handleDoubleClick } =
    useMinimapDrag(trackRef, visibleRange, onRangeChange);

  const hoveredMarker = hoveredIdx !== null ? markers[hoveredIdx] : null;

  const cursorStyle =
    dragging === 'left' || dragging === 'right' ? 'ew-resize' : 'crosshair';

  const activePct = hoverPct ?? externalHoverPct ?? null;

  return (
    <div
      ref={trackRef}
      className="group relative h-[43px] overflow-hidden rounded-lg border border-border/50 bg-muted/30"
      style={{ cursor: cursorStyle }}
      onPointerDown={handleTrackDown}
      onDoubleClick={handleDoubleClick}
      onMouseMove={(e) => {
        if (!trackRef.current) {
          return;
        }
        const pct = pctFromEvent(e, trackRef.current);
        setHoverPct(pct);
        onHoverPctChange?.(pct);
      }}
      onMouseLeave={() => {
        setHoverPct(null);
        onHoverPctChange?.(null);
      }}
    >
      <SpanMarkers
        markers={markers}
        hoveredIdx={hoveredIdx}
        onHoveredIdxChange={setHoveredIdx}
        onTooltipPosChange={setTooltipPos}
        onSpanSelect={onSpanSelect}
      />

      <RangeHandles
        visibleRange={visibleRange}
        dragging={dragging}
        startDrag={startDrag}
      />

      {activePct !== null && !dragging && (
        <CursorOverlay activePct={activePct} totalMs={totalMs} />
      )}

      {dragging && (
        <DragAnnotation visibleRange={visibleRange} totalMs={totalMs} />
      )}

      <div className="pointer-events-none absolute inset-0 z-[4] rounded-[inherit] shadow-[inset_0px_7px_10px_0px_rgba(0,0,0,0.01),inset_0px_1px_3px_0px_rgba(0,0,0,0.01)]" />

      {hoveredMarker && tooltipPos && !dragging && (
        <MinimapTooltip marker={hoveredMarker} position={tooltipPos} />
      )}
    </div>
  );
}
