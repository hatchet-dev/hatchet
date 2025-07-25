import { useEffect, useRef } from 'react';

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

  const resetTimeout = () => {
    if (timeoutRef.current) {
      clearTimeout(timeoutRef.current);
    }

    timeoutRef.current = setTimeout(() => {
      onInactive();
    }, timeoutMs);
  };

  const throttledResetTimeout = () => {
    if (throttleRef.current) {
      return; // Already throttled
    }

    throttleRef.current = setTimeout(() => {
      throttleRef.current = null;
    }, throttleMs);

    resetTimeout();
  };

  useEffect(() => {
    if (!enabled) {
      return;
    }

    // Set initial timeout
    resetTimeout();

    // Add event listeners
    events.forEach((event) => {
      document.addEventListener(event, throttledResetTimeout, true);
    });

    // Cleanup
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
  }, [enabled, timeoutMs, throttleMs, events.join(',')]); // eslint-disable-line react-hooks/exhaustive-deps

  return {};
}
