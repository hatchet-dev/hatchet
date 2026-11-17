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
 * The longest wait the SDK accepts from a server's `RetryInfo` delay, and the cap on that delay
 * as it doubles across retries. The wait blocks the caller, so a call the server asks to wait
 * longer fails at once with the server's error.
 */
export const MAX_SERVER_RETRY_DELAY = 5_000; // milliseconds

/**
 * The most attempts made at a call once one of its attempts carried a `RetryInfo` delay, the
 * same as in the other Hatchet SDKs, so a throttled call fails alike in each of them. A lower
 * configured attempt count still applies.
 */
export const MAX_THROTTLED_ATTEMPTS = 5;

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

  let attempts = retries;

  for (let i = 0; ; i++) {
    try {
      return await fn();
    } catch (e: unknown) {
      if (!shouldRetry(e)) {
        throw e;
      }
      const error = e instanceof Error ? e : new Error(String(e));

      // The throttled limit holds for the rest of the call, whatever a later attempt fails with.
      const serverDelay = retryInfoDelay(e);
      if (serverDelay > 0) {
        attempts = Math.min(attempts, MAX_THROTTLED_ATTEMPTS);
      }

      if (serverDelay > MAX_SERVER_RETRY_DELAY || i + 1 >= attempts) {
        logger.error(`Error: ${error.message}`);
        throw error;
      }

      // A throttled attempt that is retried is expected under a rate limit, so it is not an error.
      if (serverDelay > 0) {
        logger.debug(`Throttled, retrying: ${error.message}`);
      } else {
        logger.error(`Error: ${error.message}`);
      }

      // A throttled call says how long to wait; that delay replaces the client's own backoff.
      // It is doubled for each retry already made, up to the cap, which is the client behavior
      // RetryInfo documents: a server that keeps throttling repeats the same delay, so without
      // the doubling the client would not back off.
      if (serverDelay > 0) {
        await sleep(Math.min(serverDelay * 2 ** i, MAX_SERVER_RETRY_DELAY));
      } else {
        const exponentialDelay = interval * 2 ** i * 1000;
        const jitter = Math.random() * maxJitter;
        await sleep(exponentialDelay + jitter);
      }
    }
  }
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
