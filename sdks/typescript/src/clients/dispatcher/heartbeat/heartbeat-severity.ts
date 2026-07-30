import { isTransientConnectionIssue } from '@util/grpc-error';

export const MAX_MISSED_HEARTBEATS = 3;

// Transient connection issues (e.g. brief network blips, or a proxy/load
// balancer serving a plain HTTP error while the engine restarts behind it)
// are expected and shouldn't spam logs at error level. A single missed
// heartbeat isn't worth logging at all; only escalate to warn/error once it
// persists across multiple consecutive heartbeats. Mirrors the Python SDK's
// heartbeat logic.
export function classifyHeartbeatFailure(
  e: unknown,
  missedHeartbeats: number
): 'silent' | 'warn' | 'error' {
  console.info(`heartbeat ${missedHeartbeats}`);
  if (!isTransientConnectionIssue(e)) {
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
