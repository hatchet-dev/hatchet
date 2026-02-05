export function createAbortError(message = 'Operation aborted'): Error {
  const err: any = new Error(message);
  err.name = 'AbortError';
  err.code = 'ABORT_ERR';
  return err as Error;
}
