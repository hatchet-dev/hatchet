import { isAxiosError } from 'axios';

/**
 * Default TanStack Query retry policy:
 * - Don't retry most 4xx (auth/validation/permission) errors
 * - Do retry rate limiting / timeouts (429/408)
 * - Cap retries to avoid infinite loops when custom retry functions are used
 */
export function defaultQueryRetry(failureCount: number, error: unknown) {
  // Cap retries (TanStack default is 3); we keep the same upper bound.
  const withinRetryBudget = failureCount < 3;
  if (!withinRetryBudget) {
    return false;
  }

  if (isAxiosError(error)) {
    const status = error.response?.status;

    // If we didn't get a response (network error, CORS, etc), retry.
    if (!status) {
      return true;
    }

    // Don't retry most client errors (auth/forbidden/validation/etc).
    if (status >= 400 && status < 500) {
      return status === 408 || status === 429;
    }
  }

  // Retry everything else (5xx, unknown errors) within the budget.
  return true;
}
