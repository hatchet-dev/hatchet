import { setMaxListeners } from 'events';

/**
 * Attach an `abort` listener to a signal, disabling the Node.js
 * `MaxListenersExceededWarning` first.
 *
 * A single durable task can attach many concurrent listeners to the same signal
 * (fan-out children, parallel waitFor calls, etc.), easily exceeding the default
 * cap of 10. Setting max to 0 (unlimited) is safe here because every listener is
 * removed on settlement.
 *
 * Lives apart from `abort-error.ts` so the abort helpers stay free of Node imports.
 */
export function bindAbortSignalHandler(signal: AbortSignal, handler: () => void): void {
  setMaxListeners(0, signal);
  signal.addEventListener('abort', handler, { once: true });
}
