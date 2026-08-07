import { useState, useCallback, type RefObject } from 'react';

export function useCursorSync(barsRef: RefObject<HTMLDivElement | null>) {
  const [cursorPct, setCursorPct] = useState<number | null>(null);

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
    },
    [barsRef],
  );

  const onMouseLeave = useCallback(() => {
    setCursorPct(null);
  }, []);

  return { effectiveCursorPct: cursorPct, onMouseMove, onMouseLeave };
}
