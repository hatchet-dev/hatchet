import { getGrpcErrorCode, isConnectionError } from '@util/grpc-error';
import { FailureSeverity, classifyRepeatedFailure } from '@util/failure-severity';

export const MAX_TRANSIENT_LISTENER_RETRIES = 3;

export function classifyListenerFailure(e: unknown, retries: number): FailureSeverity {
  const isTransient = e === undefined || isConnectionError(getGrpcErrorCode(e));
  return classifyRepeatedFailure(isTransient, retries, MAX_TRANSIENT_LISTENER_RETRIES);
}
