/**
 * Sleeps for a given number of milliseconds without blocking the event loop
 *
 * WARNING: This is not a durable sleep. It will not be honored if the worker is
 * restarted or crashes.
 *
 * @param ms - The number of milliseconds to sleep
 * @param signal - Optional AbortSignal; if aborted, the promise rejects with Error('Cancelled').
 *                 Use in task handlers so cancellation can interrupt long sleeps.
 * @returns A promise that resolves after the given number of milliseconds (or rejects on abort)
 */
function sleep(ms: number, signal?: AbortSignal): Promise<void> {
  if (!signal) {
    return new Promise((resolve) => setTimeout(resolve, ms));
  }

  return new Promise((resolve, reject) => {
    const timer = setTimeout(() => {
      signal.removeEventListener('abort', onAbort);
      resolve();
    }, ms);

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
