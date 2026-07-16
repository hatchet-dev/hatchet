import { Logger } from './logger';
import sleep from './sleep';

export const DEFAULT_RETRY_INTERVAL = 0.1; // seconds
export const DEFAULT_RETRY_COUNT = 8;
export const DEFAULT_MAX_JITTER = 100; // milliseconds

export interface RetrierConfig {
  maxAttempts?: number;
  initialInterval?: number;
  maxJitter?: number;
  shouldRetry?: (e: unknown) => boolean;
}

export async function retrier<T>(fn: () => Promise<T>, logger: Logger, config?: RetrierConfig) {
  const retries = config?.maxAttempts ?? DEFAULT_RETRY_COUNT;
  const interval = config?.initialInterval ?? DEFAULT_RETRY_INTERVAL;
  const maxJitter = config?.maxJitter ?? DEFAULT_MAX_JITTER;
  const shouldRetry = config?.shouldRetry ?? (() => true);

  let lastError: Error | undefined;

  for (let i = 0; i < retries; i++) {
    try {
      return await fn();
    } catch (e: unknown) {
      if (!shouldRetry(e)) {
        throw e;
      }
      lastError = e instanceof Error ? e : new Error(String(e));
      logger.error(`Error: ${lastError.message}`);

      const exponentialDelay = interval * 2 ** i * 1000;
      const jitter = Math.random() * maxJitter;
      const totalDelay = exponentialDelay + jitter;

      await sleep(totalDelay);
    }
  }

  throw lastError;
}
