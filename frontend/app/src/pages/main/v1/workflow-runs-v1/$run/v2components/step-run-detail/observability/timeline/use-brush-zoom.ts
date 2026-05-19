import type { VisibleRange } from './trace-timeline-utils';
import { useState, useCallback, useRef, type RefObject } from 'react';

export type BrushRange = { lo: number; hi: number };

interface TimelineValues {
  visMinStart: number;
  timelineMaxMs: number;
  traceMinStart: number;
  traceTotalMs: number;
}

export function useBrushZoom(
  barsRef: RefObject<HTMLDivElement | null>,
  timelineValues: TimelineValues,
  onRangeChange?: (range: VisibleRange) => void,
) {
  const [brushRange, setBrushRange] = useState<BrushRange | null>(null);

  const valuesRef = useRef(timelineValues);
  valuesRef.current = timelineValues;

  const onPointerDown = useCallback(
    (e: React.PointerEvent) => {
      if (!barsRef.current || !onRangeChange) {
        return;
      }

      const rect = barsRef.current.getBoundingClientRect();
      const startPct = Math.max(
        0,
        Math.min(1, (e.clientX - rect.left) / rect.width),
      );

      const onMove = (ev: PointerEvent) => {
        if (!barsRef.current) {
          return;
        }
        const r = barsRef.current.getBoundingClientRect();
        const pct = Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width));
        const lo = Math.min(startPct, pct);
        const hi = Math.max(startPct, pct);
        if (hi - lo > 0.005) {
          setBrushRange({ lo, hi });
        }
      };

      const onUp = (ev: PointerEvent) => {
        document.removeEventListener('pointermove', onMove);
        document.removeEventListener('pointerup', onUp);

        if (!barsRef.current) {
          setBrushRange(null);
          return;
        }
        const r = barsRef.current.getBoundingClientRect();
        const pct = Math.max(0, Math.min(1, (ev.clientX - r.left) / r.width));
        const lo = Math.min(startPct, pct);
        const hi = Math.max(startPct, pct);

        setBrushRange(null);

        if (hi - lo >= 0.02) {
          const v = valuesRef.current;
          const newStartMs = v.visMinStart + v.timelineMaxMs * lo;
          const newEndMs = v.visMinStart + v.timelineMaxMs * hi;
          onRangeChange({
            startPct: Math.max(
              0,
              (newStartMs - v.traceMinStart) / v.traceTotalMs,
            ),
            endPct: Math.min(1, (newEndMs - v.traceMinStart) / v.traceTotalMs),
          });
        }
      };

      document.addEventListener('pointermove', onMove);
      document.addEventListener('pointerup', onUp);
    },
    [barsRef, onRangeChange],
  );

  const onDoubleClick = useCallback(() => {
    onRangeChange?.({ startPct: 0, endPct: 1 });
  }, [onRangeChange]);

  return { brushRange, onPointerDown, onDoubleClick };
}
