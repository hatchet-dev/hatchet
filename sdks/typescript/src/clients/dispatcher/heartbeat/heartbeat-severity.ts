import { isConnectionError } from '@util/grpc-error';
import { FailureSeverity, classifyRepeatedFailure } from '@util/failure-severity';

export const MAX_MISSED_HEARTBEATS = 3;

// determines whether to immediately error log or wait for additional errors
export function classifyHeartbeatFailure(
  code: number | undefined,
  missedHeartbeats: number
): FailureSeverity {
  return classifyRepeatedFailure(isConnectionError(code), missedHeartbeats, MAX_MISSED_HEARTBEATS);
}
