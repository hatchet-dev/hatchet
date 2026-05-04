import { useCallback, useEffect, useRef } from 'react';

interface UseInactivityDetectionOptions {
  timeoutMs?: number;
  throttleMs?: number;
  onInactive?: () => void;
}

const DEFAULT_EVENTS = [
  'mousedown',
  'mousemove',
  'keypress',
  'scroll',
  'touchstart',
  'click',
];

export function useInactivityDetection(
  options: UseInactivityDetectionOptions = {},
) {
  const { timeoutMs = -1, throttleMs = 1000, onInactive } = options;

  const enabled = timeoutMs > 0;

  // Keep onInactive callable without it being a dep — avoids resetting the
  // timer on every render when the caller passes an inline arrow function.
  const onInactiveRef = useRef(onInactive);
  useEffect(() => {
    onInactiveRef.current = onInactive;
  });

  const timeoutRef = useRef<NodeJS.Timeout | null>(null);
  const throttleRef = useRef<NodeJS.Timeout | null>(null);

  const resetTimeout = useCallback(() => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }
    timeoutRef.current = setTimeout(() => {
      onInactiveRef.current?.();
    }, timeoutMs);
  }, [timeoutMs]);

  const throttledResetTimeout = useCallback(() => {
    if (throttleRef.current) {
      return;
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

    DEFAULT_EVENTS.forEach((event) => {
      document.addEventListener(event, throttledResetTimeout, true);
    });

    return () => {
      if (timeoutRef.current) {
        clearTimeout(timeoutRef.current);
      }
      if (throttleRef.current) {
        clearTimeout(throttleRef.current);
      }
      DEFAULT_EVENTS.forEach((event) => {
        document.removeEventListener(event, throttledResetTimeout, true);
      });
    };
  }, [enabled, timeoutMs, throttleMs, resetTimeout, throttledResetTimeout]);

  return {};
}
