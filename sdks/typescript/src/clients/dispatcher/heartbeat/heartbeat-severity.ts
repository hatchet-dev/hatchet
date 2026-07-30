import { isConnectionError } from '@util/grpc-error';

export const MAX_MISSED_HEARTBEATS = 3;

// determines whether to immediately error log or wait for additional errors
export function classifyHeartbeatFailure(
  code: number | undefined,
  missedHeartbeats: number
): 'silent' | 'warn' | 'error' {
  console.info(`heartbeat ${missedHeartbeats}`);
  if (!isConnectionError(code)) {
    return 'error';
  }

  if (missedHeartbeats >= MAX_MISSED_HEARTBEATS) {
    return 'error';
  }

  if (missedHeartbeats > 1) {
    return 'warn';
  }

  return 'silent';
}
