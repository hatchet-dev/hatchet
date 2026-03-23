import { useState, useMemo, useCallback, useRef, type RefObject } from 'react';

interface TimelineValues {
  visMinStart: number;
  timelineMaxMs: number;
  traceMinStart: number;
  traceTotalMs: number;
}

export function useCursorSync(
  barsRef: RefObject<HTMLDivElement | null>,
  timelineValues: TimelineValues,
  externalCursorPct?: number | null,
  onCursorPctChange?: (pct: number | null) => void,
) {
  const [cursorPct, setCursorPct] = useState<number | null>(null);

  const valuesRef = useRef(timelineValues);
  valuesRef.current = timelineValues;

  const { visMinStart, timelineMaxMs, traceMinStart, traceTotalMs } =
    timelineValues;

  const effectiveCursorPct = useMemo(() => {
    if (cursorPct !== null) {
      return cursorPct;
    }
    if (externalCursorPct == null || timelineMaxMs <= 0) {
      return null;
    }
    const timeMs = traceMinStart + externalCursorPct * traceTotalMs;
    const localPct = (timeMs - visMinStart) / timelineMaxMs;
    if (localPct < 0 || localPct > 1) {
      return null;
    }
    return localPct;
  }, [
    cursorPct,
    externalCursorPct,
    visMinStart,
    timelineMaxMs,
    traceMinStart,
    traceTotalMs,
  ]);

  const onMouseMove = useCallback(
    (e: React.MouseEvent) => {
      if (!barsRef.current) {
        return;
      }
      const rect = barsRef.current.getBoundingClientRect();
      const localPct = Math.max(
        0,
        Math.min(1, (e.clientX - rect.left) / rect.width),
      );
      setCursorPct(localPct);
      if (onCursorPctChange) {
        const v = valuesRef.current;
        const timeMs = v.visMinStart + localPct * v.timelineMaxMs;
        const fullPct =
          v.traceTotalMs > 0 ? (timeMs - v.traceMinStart) / v.traceTotalMs : 0;
        onCursorPctChange(Math.max(0, Math.min(1, fullPct)));
      }
    },
    [barsRef, onCursorPctChange],
  );

  const onMouseLeave = useCallback(() => {
    setCursorPct(null);
    onCursorPctChange?.(null);
  }, [onCursorPctChange]);

  return { effectiveCursorPct, onMouseMove, onMouseLeave };
}
