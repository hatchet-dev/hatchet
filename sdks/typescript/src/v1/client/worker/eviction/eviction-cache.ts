import { durationToMs } from '../../duration';
import { EvictionPolicy } from './eviction-policy';

export type ActionKey = string;

export enum EvictionCause {
  TTL_EXCEEDED = 'ttl_exceeded',
  CAPACITY_PRESSURE = 'capacity_pressure',
  WORKER_SHUTDOWN = 'worker_shutdown',
}

export interface DurableRunRecord {
  key: ActionKey;
  taskRunExternalId: string;
  evictionPolicy: EvictionPolicy | undefined;
  registeredAt: number;

  waitingSince: number | undefined;
  waitKind: string | undefined;
  waitResourceId: string | undefined;
  // Ref-counted so concurrent waits (e.g. multiple child results via
  // Promise.all) don't prematurely clear the waiting flag when one
  // child completes before the others.
  _waitCount: number;

  evictionReason: string | undefined;
}

export class DurableEvictionCache {
  private _runs = new Map<ActionKey, DurableRunRecord>();

  registerRun(
    key: ActionKey,
    taskRunExternalId: string,
    now: number,
    evictionPolicy: EvictionPolicy | undefined
  ): void {
    this._runs.set(key, {
      key,
      taskRunExternalId,
      evictionPolicy,
      registeredAt: now,
      waitingSince: undefined,
      waitKind: undefined,
      waitResourceId: undefined,
      _waitCount: 0,
      evictionReason: undefined,
    });
  }

  unregisterRun(key: ActionKey): void {
    this._runs.delete(key);
  }

  get(key: ActionKey): DurableRunRecord | undefined {
    return this._runs.get(key);
  }

  getAllWaiting(): DurableRunRecord[] {
    return [...this._runs.values()].filter((r) => r._waitCount > 0);
  }

  markWaiting(key: ActionKey, now: number, waitKind: string, resourceId: string): void {
    const rec = this._runs.get(key);
    if (!rec) return;
    rec._waitCount += 1;
    if (rec._waitCount === 1) {
      rec.waitingSince = now;
    }
    rec.waitKind = waitKind;
    rec.waitResourceId = resourceId;
  }

  markActive(key: ActionKey): void {
    const rec = this._runs.get(key);
    if (!rec) return;
    rec._waitCount = Math.max(0, rec._waitCount - 1);
    if (rec._waitCount === 0) {
      rec.waitingSince = undefined;
      rec.waitKind = undefined;
      rec.waitResourceId = undefined;
    }
  }

  selectEvictionCandidate(
    now: number,
    durableSlots: number,
    reserveSlots: number,
    minWaitForCapacityEvictionMs: number
  ): ActionKey | undefined {
    const waiting = [...this._runs.values()].filter(
      (r) => r._waitCount > 0 && r.evictionPolicy !== undefined
    );

    if (waiting.length === 0) return undefined;

    const ttlEligible = waiting.filter((r) => {
      const ttl = r.evictionPolicy?.ttl;
      if (!ttl || !r.waitingSince) return false;
      return now - r.waitingSince >= durationToMs(ttl);
    });

    if (ttlEligible.length > 0) {
      ttlEligible.sort(
        (a, b) =>
          (a.evictionPolicy?.priority ?? 0) - (b.evictionPolicy?.priority ?? 0) ||
          (a.waitingSince ?? now) - (b.waitingSince ?? now)
      );
      const [chosen] = ttlEligible;
      chosen.evictionReason = buildEvictionReason(EvictionCause.TTL_EXCEEDED, chosen);
      return chosen.key;
    }

    if (!this._hasCapacityPressure(durableSlots, reserveSlots, waiting.length)) {
      return undefined;
    }

    const capacityCandidates = waiting.filter(
      (r) =>
        r.evictionPolicy?.allowCapacityEviction !== false &&
        r.waitingSince !== undefined &&
        now - r.waitingSince >= minWaitForCapacityEvictionMs
    );

    if (capacityCandidates.length === 0) return undefined;

    capacityCandidates.sort(
      (a, b) =>
        (a.evictionPolicy?.priority ?? 0) - (b.evictionPolicy?.priority ?? 0) ||
        (a.waitingSince ?? now) - (b.waitingSince ?? now)
    );
    const [chosen] = capacityCandidates;
    chosen.evictionReason = buildEvictionReason(EvictionCause.CAPACITY_PRESSURE, chosen);
    return chosen.key;
  }

  private _hasCapacityPressure(
    durableSlots: number,
    reserveSlots: number,
    waitingCount: number
  ): boolean {
    if (durableSlots <= 0) return false;
    const maxWaiting = durableSlots - reserveSlots;
    if (maxWaiting <= 0) return false;
    return waitingCount >= maxWaiting;
  }
}

export function buildEvictionReason(cause: EvictionCause, rec: DurableRunRecord): string {
  let waitDesc = rec.waitKind || 'unknown';
  if (rec.waitResourceId) {
    waitDesc = `${waitDesc}(${rec.waitResourceId})`;
  }

  switch (cause) {
    case EvictionCause.TTL_EXCEEDED: {
      const ttlStr = rec.evictionPolicy?.ttl ? ` (${rec.evictionPolicy.ttl})` : '';
      return `Wait TTL${ttlStr} exceeded while waiting on ${waitDesc}`;
    }
    case EvictionCause.CAPACITY_PRESSURE:
      return `Worker at capacity while waiting on ${waitDesc}`;
    case EvictionCause.WORKER_SHUTDOWN:
      return `Worker shutdown while waiting on ${waitDesc}`;
    default: {
      const _exhaustive: never = cause;
      throw new Error(`Unknown eviction cause: ${_exhaustive}`);
    }
  }
}
