/**
 * The `errors` an API error response carries, whichever generated client raised it: an axios
 * error holds the body under `response.data`, the fetch client throws its `HttpResponse`
 * with the body under `error`. `undefined` for any other error, or a response with no body.
 */
export function restErrors(err: unknown): unknown {
  if (typeof err !== 'object' || err === null) {
    return undefined;
  }
  if ('isAxiosError' in err) {
    const { response } = err as { response?: { data?: { errors?: unknown } } };
    return response?.data?.errors;
  }
  if ('error' in err && 'status' in err && 'ok' in err) {
    return (err as { error?: { errors?: unknown } }).error?.errors;
  }
  return undefined;
}

/**
 * Rethrows an API error response as an `Error` whose message is the response's `errors` as
 * JSON, with the original error as its cause; any other error is rethrown as is.
 */
export function rethrowRestError(err: unknown): never {
  const errors = restErrors(err);
  if (errors === undefined) {
    throw err;
  }
  throw new Error(JSON.stringify(errors), { cause: err });
}
