import { useCallback, useEffect, useRef } from 'react';

interface UseInactivityDetectionOptions {
  timeoutMs?: number;
  throttleMs?: number;
  events?: string[];
  onInactive?: () => void;
}

export function useInactivityDetection(
  options: UseInactivityDetectionOptions = {},
) {
  const {
    timeoutMs = -1, // -1 means disabled
    throttleMs = 1000, // 1 second throttle
    events = [
      'mousedown',
      'mousemove',
      'keypress',
      'scroll',
      'touchstart',
      'click',
    ],
    onInactive = () => {},
  } = options;

  const enabled = timeoutMs > 0;

  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const throttleRef = useRef<NodeJS.Timeout | null>(null);

  const resetTimeout = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      onInactive();
    }, timeoutMs);
  }, [onInactive, timeoutMs]);

  const throttledResetTimeout = useCallback(() => {
    if (throttleRef.current) {
      return; // Already throttled
    }

    throttleRef.current = setTimeout(() => {
      throttleRef.current = null;
    }, throttleMs);

    resetTimeout();
  }, [throttleMs, resetTimeout]);

  useEffect(() => {
    if (!enabled) {
      return;
    }

    resetTimeout();

    events.forEach((event) => {
      document.addEventListener(event, throttledResetTimeout, true);
    });

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      if (throttleRef.current) {
        clearTimeout(throttleRef.current);
      }
      events.forEach((event) => {
        document.removeEventListener(event, throttledResetTimeout, true);
      });
    };
  }, [
    enabled,
    timeoutMs,
    throttleMs,
    resetTimeout,
    events,
    throttledResetTimeout,
  ]);

  return {};
}
