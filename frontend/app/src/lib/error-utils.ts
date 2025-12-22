import { isAxiosError } from 'axios';

function getUnknownProp(value: unknown, key: string): unknown {
  if (!value || typeof value !== 'object') {
    return undefined;
  }

  return Reflect.get(value, key);
}

export function getErrorStatus(error: unknown): number | undefined {
  if (!error) {
    return undefined;
  }

  // Route loaders often throw a Response
  if (error instanceof Response) {
    return error.status;
  }

  // Axios errors
  if (isAxiosError(error)) {
    return error.response?.status;
  }

  // TanStack Router can throw objects like { status, statusText }
  const status = getUnknownProp(error, 'status');
  return typeof status === 'number' ? status : undefined;
}

export function getErrorStatusText(error: unknown): string | undefined {
  if (!error) {
    return undefined;
  }

  if (error instanceof Response) {
    return error.statusText;
  }

  if (isAxiosError(error)) {
    return error.response?.statusText;
  }

  const statusText = getUnknownProp(error, 'statusText');
  return typeof statusText === 'string' ? statusText : undefined;
}

/**
 * React Query `retry` helper: don't retry for client errors (4xx).
 * Retries are left enabled for network errors and server errors (5xx).
 */
export function shouldRetryQueryError(error: unknown): boolean {
  const status = getErrorStatus(error);

  if (status !== undefined && status >= 400 && status < 500) {
    return false;
  }

  return true;
}
