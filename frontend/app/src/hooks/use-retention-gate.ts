import {
  isBeforeRetention,
  isTimeWindowOutsideRetention,
  type TimeWindowPreset,
} from '@/lib/utils/retention';
import { useCallback, useState } from 'react';

export type RetentionAttempt =
  | { kind: 'preset'; window: TimeWindowPreset }
  | { kind: 'since'; date: Date };

export function useRetentionGate(period?: string) {
  const [attempt, setAttempt] = useState<RetentionAttempt | null>(null);

  const close = useCallback(() => setAttempt(null), []);

  const blockSince = useCallback((date: Date) => {
    setAttempt({ kind: 'since', date });
  }, []);

  const tryTimeWindow = useCallback(
    (timeWindow: string, apply: () => void) => {
      if (
        period &&
        timeWindow !== 'custom' &&
        isTimeWindowOutsideRetention(timeWindow, period)
      ) {
        setAttempt({ kind: 'preset', window: timeWindow as TimeWindowPreset });
        return false;
      }
      apply();
      return true;
    },
    [period],
  );

  const trySince = useCallback(
    (iso: string | undefined, apply: () => void) => {
      if (iso && period && isBeforeRetention(iso, period)) {
        setAttempt({ kind: 'since', date: new Date(iso) });
        return false;
      }
      apply();
      return true;
    },
    [period],
  );

  return {
    attempt,
    close,
    blockSince,
    tryTimeWindow,
    trySince,
    period,
  };
}
