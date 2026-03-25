import { useEffect, useState } from 'react';

const TICK_INTERVAL_MS = 30;

export function useLiveClock(enabled: boolean): number {
  const [now, setNow] = useState(Date.now);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    setNow(Date.now());
    const id = setInterval(() => setNow(Date.now()), TICK_INTERVAL_MS);
    return () => clearInterval(id);
  }, [enabled]);

  return now;
}
