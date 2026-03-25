import { MIN_RANGE_PCT } from './minimap-types';
import type { DragMode, DragState, TimeRange } from './minimap-types';
import { pctFromEvent } from './minimap-utils';
import { useCallback, useEffect, useRef, useState } from 'react';

export function useMinimapDrag(
  trackRef: React.RefObject<HTMLDivElement | null>,
  visibleRange: TimeRange,
  onRangeChange: (range: TimeRange) => void,
) {
  const [dragging, setDragging] = useState<DragMode>(null);
  const dragRef = useRef<DragState | null>(null);

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
    [trackRef, startDrag, onRangeChange],
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
  }, [dragging, trackRef, onRangeChange]);

  return { dragging, startDrag, handleTrackDown, handleDoubleClick };
}
