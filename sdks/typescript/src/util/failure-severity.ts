export type FailureSeverity = 'silent' | 'warn' | 'error';

export function classifyRepeatedFailure(
  isTransient: boolean,
  attempt: number,
  threshold: number
): FailureSeverity {
  if (!isTransient) {
    return 'error';
  }

  if (attempt >= threshold) {
    return 'error';
  }

  if (attempt > 1) {
    return 'warn';
  }

  return 'silent';
}
