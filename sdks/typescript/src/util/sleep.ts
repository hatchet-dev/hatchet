import { Duration, durationToMs } from '../v1/client/duration';

/**
 * Sleeps for a given number of milliseconds without blocking the event loop
 *
 * WARNING: This is not a durable sleep. It will not be honored if the worker is
 * restarted or crashes.
 *
 * @param duration - The number of milliseconds to sleep, or a Duration (e.g. "5s", \{ seconds: 5 \})
 * @param signal - Optional AbortSignal; if aborted, the promise rejects with Error('Cancelled').
 *                 Use in task handlers so cancellation can interrupt long sleeps.
 * @returns A promise that resolves after the given number of milliseconds (or rejects on abort)
 */
function sleep(duration: number | Duration, signal?: AbortSignal): Promise<void> {
  const timeout = typeof duration === 'number' ? duration : durationToMs(duration);
  if (!signal) {
    return new Promise((resolve) => {
      setTimeout(resolve, timeout);
    });
  }

  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', onAbort);
      resolve();
    }, timeout);

    const onAbort = () => {
      clearTimeout(timer);
      reject(new Error('Cancelled'));
    };

    if (signal.aborted) {
      clearTimeout(timer);
      reject(new Error('Cancelled'));
      return;
    }

    signal.addEventListener('abort', onAbort, { once: true });
  });
}

export default sleep;
