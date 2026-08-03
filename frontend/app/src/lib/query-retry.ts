import { isAxiosError } from 'axios';

/**
 * Base class for errors that `defaultQueryRetry` must not retry, e.g. because
 * the layer that threw them already exhausted its own retry budget.
 */
export class NonRetryableError extends Error {}

/**
 * Whether a failed HTTP request is worth retrying:
 * - No response (network error, CORS, timeout): retry
 * - Most 4xx (auth/validation/permission) won't be fixed by retrying; 408/429 will
 * - 5xx and unknown errors: retry
 */
export function isRetryableRequestError(error: unknown): boolean {
  if (isAxiosError(error)) {
    const status = error.response?.status;

    if (!status) {
      return true;
    }

    if (status >= 400 && status < 500) {
      return status === 408 || status === 429;
    }
  }

  return true;
}

/**
 * Default TanStack Query retry policy: retry retryable request errors, capped
 * at the same upper bound as the TanStack default (3).
 */
export function defaultQueryRetry(failureCount: number, error: unknown) {
  if (error instanceof NonRetryableError) {
    return false;
  }

  if (failureCount >= 3) {
    return false;
  }

  return isRetryableRequestError(error);
}
