import { useEffect, useState } from 'react';

/**
 * The current time in milliseconds, refreshed every intervalMs, for memoized values that
 * depend on how long ago something happened and would otherwise go stale while their other
 * inputs stay the same.
 */
export function useNow(intervalMs: number): number {
  const [now, setNow] = useState(() => Date.now());

  useEffect(() => {
    const id = setInterval(() => setNow(Date.now()), intervalMs);

    return () => clearInterval(id);
  }, [intervalMs]);

  return now;
}
