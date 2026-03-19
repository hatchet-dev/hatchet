import type { OtelSpanTree } from '@/components/v1/agent-prism/span-tree-type';
import { OtelStatusCode } from '@/lib/api/generated/data-contracts';
import { cn } from '@/lib/utils';
import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

const MIN_RANGE_PCT = 0.05;

type TimeRange = { startPct: number; endPct: number };
type DragMode = 'left' | 'right' | 'brush' | null;

type SpanMarker = {
  pct: number;
  statusCode: OtelStatusCode;
  hasErrorInTree: boolean;
  inProgress: boolean;
  spanName: string;
  durationMs: number;
  visible: boolean;
  span: OtelSpanTree;
  ancestorSpanIds: string[];
};

type DragState = {
  mouseX: number;
  startPct: number;
  endPct: number;
  anchorPct?: number;
};

function getHatchetDisplayName(span: OtelSpanTree): string {
  if (!span.spanName.startsWith('hatchet.')) {
    return span.spanName;
  }
  if (span.spanAttributes?.['hatchet.step_name']) {
    return span.spanAttributes['hatchet.step_name'];
  }
  if (span.spanAttributes?.['hatchet.workflow_name']) {
    return span.spanAttributes['hatchet.workflow_name'];
  }
  const actionId = span.spanAttributes?.['hatchet.action_id'];
  if (actionId?.includes(':')) {
    return actionId.split(':')[0];
  }
  return span.spanName;
}

function hasErrorInTree(span: OtelSpanTree): boolean {
  if (span.statusCode === OtelStatusCode.ERROR) {
    return true;
  }
  return span.children.some(hasErrorInTree);
}

function getMarkerColor(marker: SpanMarker): string {
  if (marker.inProgress) {
    return 'bg-blue-500';
  }
  if (marker.hasErrorInTree) {
    return 'bg-red-500';
  }
  return 'bg-green-500';
}

function getDotColor(marker: SpanMarker): string {
  if (marker.inProgress) {
    return 'bg-blue-500';
  }
  if (marker.hasErrorInTree) {
    return 'bg-red-500';
  }
  return 'bg-green-500';
}

function formatDuration(ms: number): string {
  if (ms < 1) {
    return '<1ms';
  }
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }
  return `${(ms / 1000).toFixed(1)}s`;
}

function formatOffset(ms: number): string {
  if (ms <= 0) {
    return '0ms';
  }
  if (ms < 1000) {
    return `${Math.round(ms)}ms`;
  }
  if (ms < 60000) {
    return `${(ms / 1000).toFixed(1)}s`;
  }
  const m = Math.floor(ms / 60000);
  const s = Math.floor((ms % 60000) / 1000);
  return s > 0 ? `${m}m${s}s` : `${m}m`;
}

function collectSpanMarkers(
  trees: OtelSpanTree[],
  minMs: number,
  totalMs: number,
  expandedIds?: Set<string>,
): SpanMarker[] {
  const markers: SpanMarker[] = [];

  const traverse = (
    node: OtelSpanTree,
    parentVisible: boolean,
    ancestors: string[],
  ) => {
    const startMs = new Date(node.createdAt).getTime();
    const pct = totalMs > 0 ? (startMs - minMs) / totalMs : 0;
    markers.push({
      pct: Math.max(0, Math.min(1, pct)),
      statusCode: node.statusCode,
      hasErrorInTree: hasErrorInTree(node),
      inProgress: !!node.inProgress,
      spanName: getHatchetDisplayName(node),
      durationMs: node.durationNs / 1_000_000,
      visible: parentVisible,
      span: node,
      ancestorSpanIds: ancestors,
    });
    const childrenVisible =
      parentVisible && (!expandedIds || expandedIds.has(node.spanId));
    const nextAncestors = [...ancestors, node.spanId];
    node.children?.forEach((child) =>
      traverse(child, childrenVisible, nextAncestors),
    );
  };

  trees.forEach((tree) => traverse(tree, true, []));
  return markers.sort((a, b) => a.pct - b.pct);
}

function pctFromEvent(e: { clientX: number }, el: HTMLElement): number {
  const rect = el.getBoundingClientRect();
  return Math.max(0, Math.min(1, (e.clientX - rect.left) / rect.width));
}

interface TraceMinimapProps {
  spanTrees: OtelSpanTree[];
  minMs: number;
  maxMs: number;
  visibleRange: TimeRange;
  onRangeChange: (range: TimeRange) => void;
  expandedSpanIds?: string[];
  onSpanSelect?: (span: OtelSpanTree, ancestorSpanIds: string[]) => void;
}

export function TraceMinimap({
  spanTrees,
  minMs,
  maxMs,
  visibleRange,
  onRangeChange,
  expandedSpanIds,
  onSpanSelect,
}: TraceMinimapProps) {
  const trackRef = useRef<HTMLDivElement>(null);
  const [dragging, setDragging] = useState<DragMode>(null);
  const dragRef = useRef<DragState | null>(null);

  const [hoveredIdx, setHoveredIdx] = useState<number | null>(null);
  const [tooltipPos, setTooltipPos] = useState<{
    x: number;
    y: number;
  } | null>(null);
  const [hoverPct, setHoverPct] = useState<number | null>(null);

  const expandedSet = useMemo(
    () => (expandedSpanIds ? new Set(expandedSpanIds) : undefined),
    [expandedSpanIds],
  );

  const totalMs = maxMs - minMs;
  const markers = collectSpanMarkers(spanTrees, minMs, totalMs, expandedSet);

  const startDrag = useCallback(
    (mode: DragMode, e: React.PointerEvent, anchorPct?: number) => {
      e.preventDefault();
      e.stopPropagation();
      setDragging(mode);
      dragRef.current = {
        mouseX: e.clientX,
        startPct: visibleRange.startPct,
        endPct: visibleRange.endPct,
        anchorPct,
      };
    },
    [visibleRange],
  );

  const handleTrackDown = useCallback(
    (e: React.PointerEvent) => {
      if (!trackRef.current) {
        return;
      }
      const clickPct = pctFromEvent(e, trackRef.current);
      onRangeChange({
        startPct: clickPct,
        endPct: Math.min(1, clickPct + MIN_RANGE_PCT),
      });
      startDrag('brush', e, clickPct);
    },
    [startDrag, onRangeChange],
  );

  const handleDoubleClick = useCallback(() => {
    onRangeChange({ startPct: 0, endPct: 1 });
  }, [onRangeChange]);

  useEffect(() => {
    if (!dragging) {
      return;
    }

    const onMove = (e: PointerEvent) => {
      if (!dragRef.current || !trackRef.current) {
        return;
      }

      const trackWidth = trackRef.current.getBoundingClientRect().width;
      if (trackWidth <= 0) {
        return;
      }

      const { startPct: origStart, endPct: origEnd } = dragRef.current;

      if (dragging === 'brush') {
        const anchor = dragRef.current.anchorPct!;
        const current = pctFromEvent(e, trackRef.current);
        let lo = Math.min(anchor, current);
        let hi = Math.max(anchor, current);
        if (hi - lo < MIN_RANGE_PCT) {
          if (current >= anchor) {
            hi = Math.min(1, lo + MIN_RANGE_PCT);
          } else {
            lo = Math.max(0, hi - MIN_RANGE_PCT);
          }
        }
        onRangeChange({ startPct: lo, endPct: hi });
        return;
      }

      const deltaPct = (e.clientX - dragRef.current.mouseX) / trackWidth;

      if (dragging === 'left') {
        const newStart = Math.max(
          0,
          Math.min(origEnd - MIN_RANGE_PCT, origStart + deltaPct),
        );
        onRangeChange({ startPct: newStart, endPct: origEnd });
      } else if (dragging === 'right') {
        const newEnd = Math.min(
          1,
          Math.max(origStart + MIN_RANGE_PCT, origEnd + deltaPct),
        );
        onRangeChange({ startPct: origStart, endPct: newEnd });
      }
    };

    const onUp = () => {
      setDragging(null);
      dragRef.current = null;
    };

    document.addEventListener('pointermove', onMove);
    document.addEventListener('pointerup', onUp);
    return () => {
      document.removeEventListener('pointermove', onMove);
      document.removeEventListener('pointerup', onUp);
    };
  }, [dragging, onRangeChange]);

  const sPct = visibleRange.startPct * 100;
  const ePct = visibleRange.endPct * 100;

  const hoveredMarker = hoveredIdx !== null ? markers[hoveredIdx] : null;

  const cursorStyle =
    dragging === 'left' || dragging === 'right' ? 'ew-resize' : 'crosshair';

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
        setHoverPct(pctFromEvent(e, trackRef.current));
      }}
      onMouseLeave={() => setHoverPct(null)}
    >
      {/* Event markers */}
      {markers.map((m, i) => (
        <div
          key={i}
          className={cn(
            'absolute inset-y-[6px] flex cursor-pointer flex-col justify-center transition-[transform,opacity]',
            m.hasErrorInTree ? 'z-[3]' : 'z-[2]',
            hoveredIdx === i && 'z-[5] scale-x-150',
            !m.visible && 'opacity-[0.01]',
          )}
          style={{ left: `${m.pct * 100}%`, width: 6 }}
          onPointerDown={(e) => {
            if (onSpanSelect) {
              e.stopPropagation();
            }
          }}
          onClick={() => {
            onSpanSelect?.(m.span, m.ancestorSpanIds);
          }}
          onMouseEnter={(e) => {
            setHoveredIdx(i);
            setTooltipPos({ x: e.clientX, y: e.clientY });
          }}
          onMouseMove={(e) => {
            setTooltipPos({ x: e.clientX, y: e.clientY });
          }}
          onMouseLeave={() => {
            setHoveredIdx(null);
            setTooltipPos(null);
          }}
        >
          <div className={cn('flex-1 rounded-full', getMarkerColor(m))} />
        </div>
      ))}

      {/* Left dim overlay */}
      {dragging !== 'brush' && (
        <div
          className="pointer-events-none absolute inset-y-0 left-0 z-[1] bg-background/70"
          style={{ width: `${sPct}%` }}
        />
      )}

      {/* Right dim overlay */}
      {dragging !== 'brush' && (
        <div
          className="pointer-events-none absolute inset-y-0 right-0 z-[1] bg-background/70"
          style={{ width: `${100 - ePct}%` }}
        />
      )}

      {/* Selected region with handles inside — visible on hover or while dragging */}
      <div
        className={cn(
          'pointer-events-none absolute inset-y-0 z-[3] transition-opacity duration-150',
          dragging === 'brush'
            ? 'opacity-0'
            : dragging
              ? 'opacity-100'
              : 'opacity-0 group-hover:opacity-100',
        )}
        style={{ left: `${sPct}%`, right: `${100 - ePct}%` }}
      >
        {/* Left handle — pinned to left edge of selected region */}
        <div
          className="pointer-events-auto absolute bottom-0 left-0 top-0 flex w-[17px] flex-col items-center justify-center"
          style={{ cursor: 'ew-resize' }}
          onPointerDown={(e) => startDrag('left', e)}
        >
          <div className="flex h-[6px] items-center justify-center">
            <div className="h-px w-1.5 bg-border" />
          </div>
          <div className="flex flex-1 items-center justify-center rounded-md border border-border/60 bg-muted/80 px-0.5">
            <svg
              width="8"
              height="12"
              viewBox="0 0 8 12"
              fill="none"
              className="text-muted-foreground"
            >
              <path
                d="M5 1L1 6L5 11"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>
          <div className="flex h-[6px] items-center justify-center">
            <div className="h-px w-1.5 bg-border" />
          </div>
        </div>

        {/* Right handle — pinned to right edge of selected region */}
        <div
          className="pointer-events-auto absolute bottom-0 right-0 top-0 flex w-[17px] flex-col items-center justify-center"
          style={{ cursor: 'ew-resize' }}
          onPointerDown={(e) => startDrag('right', e)}
        >
          <div className="flex h-[6px] items-center justify-center">
            <div className="h-px w-1.5 bg-border" />
          </div>
          <div className="flex flex-1 items-center justify-center rounded-md border border-border/60 bg-muted/80 px-0.5">
            <svg
              width="8"
              height="12"
              viewBox="0 0 8 12"
              fill="none"
              className="text-muted-foreground"
            >
              <path
                d="M3 1L7 6L3 11"
                stroke="currentColor"
                strokeWidth="1.5"
                strokeLinecap="round"
                strokeLinejoin="round"
              />
            </svg>
          </div>
          <div className="flex h-[6px] items-center justify-center">
            <div className="h-px w-1.5 bg-border" />
          </div>
        </div>
      </div>

      {/* Hover cursor line + timestamp */}
      {hoverPct !== null && !dragging && (
        <>
          <div
            className="pointer-events-none absolute inset-y-0 z-[5] w-px bg-foreground/60"
            style={{ left: `${hoverPct * 100}%` }}
          />
          <div
            className="pointer-events-none absolute top-0.5 z-[7] whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background"
            style={{
              left: `${hoverPct * 100}%`,
              transform:
                hoverPct < 0.08
                  ? 'none'
                  : hoverPct > 0.92
                    ? 'translateX(-100%)'
                    : 'translateX(-50%)',
            }}
          >
            {formatOffset(totalMs * hoverPct)}
          </div>
        </>
      )}

      {/* Drag boundary lines + timestamps */}
      {dragging && (
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
              top: '50%',
              transform: 'translateY(-50%)',
            }}
          >
            <span className="shrink-0 whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background">
              {formatOffset(totalMs * visibleRange.startPct)}
            </span>
            <div className="flex min-w-1 flex-1 items-center">
              <svg width="5" height="6" viewBox="0 0 5 6" className="shrink-0 fill-primary">
                <path d="M5 0L0 3L5 6Z" />
              </svg>
              <div className="h-px flex-1 bg-primary" />
            </div>
            <span className="shrink-0 whitespace-nowrap rounded bg-primary px-1.5 py-0.5 font-mono text-[10px] font-medium leading-tight text-primary-foreground">
              {formatOffset(totalMs * (visibleRange.endPct - visibleRange.startPct))}
            </span>
            <div className="flex min-w-1 flex-1 items-center">
              <div className="h-px flex-1 bg-primary" />
              <svg width="5" height="6" viewBox="0 0 5 6" className="shrink-0 fill-primary">
                <path d="M0 0L5 3L0 6Z" />
              </svg>
            </div>
            <span className="shrink-0 whitespace-nowrap rounded bg-foreground/90 px-1 py-px font-mono text-[10px] leading-tight text-background">
              {formatOffset(totalMs * visibleRange.endPct)}
            </span>
          </div>
        </>
      )}

      {/* Inner shadow */}
      <div className="pointer-events-none absolute inset-0 z-[4] rounded-[inherit] shadow-[inset_0px_7px_10px_0px_rgba(0,0,0,0.01),inset_0px_1px_3px_0px_rgba(0,0,0,0.01)]" />

      {/* Hover tooltip via portal */}
      {hoveredMarker &&
        tooltipPos &&
        !dragging &&
        createPortal(
          <div
            className="z-50 overflow-hidden rounded-lg border border-border bg-popover py-1 shadow-lg"
            style={{
              position: 'fixed',
              left: Math.min(tooltipPos.x + 12, window.innerWidth - 220),
              top: tooltipPos.y + 16,
              minWidth: 180,
              pointerEvents: 'none',
            }}
          >
            <div className="truncate px-3 py-1.5 font-mono text-xs text-foreground">
              {hoveredMarker.spanName}
            </div>
            <div className="flex items-center gap-2 px-3 py-1.5">
              <span
                className={cn(
                  'size-2 shrink-0 rounded-full',
                  getDotColor(hoveredMarker),
                )}
              />
              <span className="flex-1 font-mono text-xs text-muted-foreground">
                {hoveredMarker.inProgress
                  ? 'In Progress'
                  : hoveredMarker.hasErrorInTree
                    ? 'Error'
                    : 'OK'}
              </span>
              <span className="font-mono text-xs text-foreground">
                {formatDuration(hoveredMarker.durationMs)}
              </span>
            </div>
          </div>,
          document.body,
        )}
    </div>
  );
}
