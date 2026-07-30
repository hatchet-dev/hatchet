import { getGrpcErrorCode, isConnectionError } from '@util/grpc-error';

export const MAX_TRANSIENT_LISTENER_RETRIES = 3;


export function classifyListenerFailure(e: unknown, retries: number): 'warn' | 'error' {
  if (!isConnectionError(getGrpcErrorCode(e))) {
    return 'error';
  }

  return retries <= MAX_TRANSIENT_LISTENER_RETRIES ? 'warn' : 'error';
}
