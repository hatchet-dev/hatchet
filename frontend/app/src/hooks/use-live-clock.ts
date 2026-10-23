import { useEffect, useState } from 'react';

const DEFAULT_TICK_INTERVAL_MS = 30;

export function useLiveClock(
  enabled: boolean,
  tickIntervalMs = DEFAULT_TICK_INTERVAL_MS,
): number {
  const [now, setNow] = useState(Date.now);

  useEffect(() => {
    if (!enabled) {
      return;
    }
    setNow(Date.now());
    const id = setInterval(() => setNow(Date.now()), tickIntervalMs);
    return () => clearInterval(id);
  }, [enabled, tickIntervalMs]);

  return now;
}
