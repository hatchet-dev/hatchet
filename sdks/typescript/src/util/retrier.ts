import { Logger } from './logger';
import sleep from './sleep';

const DEFAULT_RETRY_INTERVAL = 0.1; // seconds
const DEFAULT_RETRY_COUNT = 8;
const MAX_JITTER = 100; // milliseconds

export async function retrier<T>(
  fn: () => Promise<T>,
  logger: Logger,
  retries: number = DEFAULT_RETRY_COUNT,
  interval: number = DEFAULT_RETRY_INTERVAL
) {
  let lastError: Error | undefined;

  // eslint-disable-next-line no-plusplus
  for (let i = 0; i < retries; i++) {
    try {
      return await fn();
    } catch (e: any) {
      lastError = e;
      logger.error(`Error: ${e.message}`);

      // Calculate exponential backoff with random jitter
      const exponentialDelay = interval * 2 ** i * 1000;
      const jitter = Math.random() * MAX_JITTER;
      const totalDelay = exponentialDelay + jitter;

      await sleep(totalDelay);
    }
  }

  throw lastError;
}
