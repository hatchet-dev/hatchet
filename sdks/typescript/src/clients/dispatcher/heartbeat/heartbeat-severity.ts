import { getGrpcErrorCode, isConnectionError, isTimeoutOrAbortError } from '@util/grpc-error';
import { FailureSeverity, classifyRepeatedFailure } from '@util/failure-severity';

export const MAX_MISSED_HEARTBEATS = 3;

// determines whether to immediately error log or wait for additional errors
export function classifyHeartbeatFailure(e: unknown, missedHeartbeats: number): FailureSeverity {
  const code = getGrpcErrorCode(e);
  const isTransient = isConnectionError(code) || isTimeoutOrAbortError(e);
  return classifyRepeatedFailure(isTransient, missedHeartbeats, MAX_MISSED_HEARTBEATS);
}
