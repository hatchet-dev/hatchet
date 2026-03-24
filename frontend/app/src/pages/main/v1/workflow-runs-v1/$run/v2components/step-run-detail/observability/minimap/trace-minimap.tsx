import { computeTimeTicks } from '../timeline/trace-timeline-utils';
import { formatDuration } from '../utils/format-utils';
import { isQueuedOnlyRoot } from '../utils/span-tree-utils';
import { useLiveClock } from '../utils/use-live-clock';
import { CursorOverlay } from './cursor-overlay';
import { DragAnnotation } from './drag-annotation';
import { MinimapTooltip } from './minimap-tooltip';
import type { TraceMinimapProps } from './minimap-types';
import { collectSpanMarkers, pctFromEvent } from './minimap-utils';
import { RangeHandles } from './range-handles';
import { SpanMarkers } from './span-markers';
import { useMinimapDrag } from './use-minimap-drag';
import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { useMemo, useRef, useState } from 'react';

export type { TimeRange } from './minimap-types';

function hasLiveProgress(nodes: OtelSpanTree[]): boolean {
  return nodes.some(
    (n) => n.inProgress || isQueuedOnlyRoot(n) || hasLiveProgress(n.children),
  );
}

export function TraceMinimap({
  spanTrees,
  minMs,
  maxMs,
  isRunning,
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

  const hasLive = useMemo(() => hasLiveProgress(spanTrees), [spanTrees]);
  const now = useLiveClock(!!isRunning && hasLive);
  const effectiveMaxMs = hasLive ? Math.max(maxMs, now) : maxMs;
  const totalMs = effectiveMaxMs - minMs;
  const markers = useMemo(
    () => collectSpanMarkers(spanTrees, minMs, totalMs, expandedSpanIds),
    [spanTrees, minMs, totalMs, expandedSpanIds],
  );

  const ticks = useMemo(() => computeTimeTicks(totalMs).ticks, [totalMs]);

  const { dragging, startDrag, handleTrackDown, handleDoubleClick } =
    useMinimapDrag(trackRef, visibleRange, onRangeChange);

  const hoveredMarker = hoveredIdx !== null ? markers[hoveredIdx] : null;

  const cursorStyle =
    dragging === 'left' || dragging === 'right' ? 'ew-resize' : 'crosshair';

  const activePct = hoverPct ?? externalHoverPct ?? null;

  return (
    <div>
      <div className="relative h-5">
        {ticks.map((t) => {
          if (t >= totalMs) {
            return null;
          }
          return (
            <div
              key={t}
              className="absolute flex h-full items-center"
              style={{
                left: `${totalMs > 0 ? (t / totalMs) * 100 : 0}%`,
              }}
            >
              <span className="whitespace-nowrap font-mono text-xs uppercase tracking-wider text-muted-foreground">
                {formatDuration(t)}
              </span>
            </div>
          );
        })}
        <div className="absolute right-0 z-10 flex h-full items-center">
          <span className="whitespace-nowrap rounded-sm bg-background px-1 font-mono text-xs uppercase tracking-wider text-muted-foreground">
            {formatDuration(totalMs)}
          </span>
        </div>
      </div>

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
    </div>
  );
}
