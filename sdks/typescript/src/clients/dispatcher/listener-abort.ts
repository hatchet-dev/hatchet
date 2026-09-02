import { isAbortError } from 'abort-controller-x';
import { getErrorMessage } from '@util/errors/hatchet-error';

export const LISTENER_SHUTDOWN_REASON = 'Worker stopping';
export const LISTENER_RECONNECT_REASON = 'Listener reconnect';

export function isListenerReconnectAbort(e: unknown): boolean {
  return isAbortError(e) && getErrorMessage(e).includes(LISTENER_RECONNECT_REASON);
}

export function isListenerShutdownAbort(e: unknown): boolean {
  return isAbortError(e) && getErrorMessage(e).includes(LISTENER_SHUTDOWN_REASON);
}
