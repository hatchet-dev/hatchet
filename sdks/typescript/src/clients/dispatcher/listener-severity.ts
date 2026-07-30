import { getGrpcErrorCode, isConnectionError } from '@util/grpc-error';

export const MAX_TRANSIENT_LISTENER_RETRIES = 3;

// A transient connection issue (the engine restarting) is expected and only
// worth a warning while the listener is still within its first few reconnect
// attempts. If it persists past that, escalate to error so a sustained outage
// doesn't retry silently forever.
export function classifyListenerFailure(e: unknown, retries: number): 'warn' | 'error' {
  if (!isConnectionError(getGrpcErrorCode(e))) {
    return 'error';
  }

  return retries <= MAX_TRANSIENT_LISTENER_RETRIES ? 'warn' : 'error';
}
