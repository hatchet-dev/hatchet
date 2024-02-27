import { Logger } from './logger';
import sleep from './sleep';

const DEFAULT_RETRY_INTERVAL = 5;
const DEFAULT_RETRY_COUNT = 5;

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
      await sleep(interval * 1000);
    }
  }

  throw lastError;
}
