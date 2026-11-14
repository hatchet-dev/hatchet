import { fromBinary } from '@bufbuild/protobuf';
import { BinaryReader, WireType } from '@bufbuild/protobuf/wire';
import { DurationSchema } from '@bufbuild/protobuf/wkt';
import { ConnectError } from '@connectrpc/connect';
import { Logger } from './logger';
import sleep from './sleep';

export const DEFAULT_RETRY_INTERVAL = 0.1; // seconds
export const DEFAULT_RETRY_COUNT = 8;
export const DEFAULT_MAX_JITTER = 100; // milliseconds

/**
 * The longest wait a server's `RetryInfo` delay can ask for before one retry. With the default
 * attempt count a call that is throttled on every attempt gives up after at most 35 seconds.
 */
export const MAX_SERVER_RETRY_DELAY = 5_000; // milliseconds

export interface RetrierConfig {
  maxAttempts?: number;
  initialInterval?: number;
  maxJitter?: number;
  shouldRetry?: (e: unknown) => boolean;
}

export async function retrier<T>(fn: () => Promise<T>, logger: Logger, config?: RetrierConfig) {
  const retries = Math.max(1, config?.maxAttempts ?? DEFAULT_RETRY_COUNT);
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

      if (i < retries - 1) {
        // A throttled call says how long to wait; waiting it replaces the client's own backoff.
        const serverDelay = retryInfoDelay(e);
        if (serverDelay > 0) {
          await sleep(Math.min(serverDelay, MAX_SERVER_RETRY_DELAY));
        } else {
          const exponentialDelay = interval * 2 ** i * 1000;
          const jitter = Math.random() * maxJitter;
          await sleep(exponentialDelay + jitter);
        }
      }
    }
  }

  throw lastError;
}

/**
 * The delay in milliseconds that a `google.rpc.RetryInfo` detail on a gRPC error asks for, or
 * zero. Connect decodes the error's `grpc-status-details-bin` trailer into details and keeps
 * each one it has no schema for as its type name and bytes. RetryInfo's only field is
 * `retry_delay` (field 1, a `google.protobuf.Duration`), so it is read from those bytes directly.
 */
function retryInfoDelay(e: unknown): number {
  if (!(e instanceof ConnectError)) {
    return 0;
  }
  for (const detail of e.details) {
    if ('type' in detail && detail.type === 'google.rpc.RetryInfo') {
      const reader = new BinaryReader(detail.value);
      while (reader.pos < reader.len) {
        const [field, wireType] = reader.tag();
        if (field === 1 && wireType === WireType.LengthDelimited) {
          const delay = fromBinary(DurationSchema, reader.bytes());
          return Number(delay.seconds) * 1000 + delay.nanos / 1_000_000;
        }
        reader.skip(wireType);
      }
    }
  }
  return 0;
}
