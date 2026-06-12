import type { DurableTaskEventLogEntryResult, PendingCallbackKey } from './durable-listener-client';

// How long the ordered-release gate stays closed waiting for a woken
// continuation to park (register its next awaited entry) before being forced
// open with a warning.
export const DEFAULT_PARK_TIMEOUT_MS = 5_000;

// How long a hole in the satisfied-order sequence may persist (while later
// completions are held) before the invocation's waiters are failed with a
// non-determinism error.
export const DEFAULT_GAP_TIMEOUT_MS = 60_000;

/**
 * Serializes the release of ordered entryCompleted responses for a single
 * durable task invocation. Completions are released to user code in
 * satisfiedOrder; after a release wakes a parked continuation, further
 * releases are held until that continuation parks again (registers its next
 * awaited entry), or the park timeout elapses.
 */
export interface OrderedReleaseGate {
  held: Map<number, { key: PendingCallbackKey; result: DurableTaskEventLogEntryResult }>;
  /** highest satisfied order released so far */
  released: number;
  /**
   * continuations woken by a gated release which have not yet parked; the
   * gate is open iff wakes === 0
   */
  wakes: number;
  /** when wakes last transitioned from zero, for the park timeout */
  wakeSince: number;
  /** when the gate first became blocked on a missing order, or null */
  gapSince: number | null;
}
