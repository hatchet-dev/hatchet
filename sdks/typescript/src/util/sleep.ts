/**
 * Sleeps for a given number of milliseconds without blocking the event loop
 * WARNING: This is not a durable sleep. It will not be honored if the worker is
 * restarted or crashes.
 * @param ms - The number of milliseconds to sleep
 * @returns A promise that resolves after the given number of milliseconds
 */
const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

export default sleep;
