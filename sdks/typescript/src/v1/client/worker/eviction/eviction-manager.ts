import { Logger } from '@hatchet/util/logger';
import { EvictionPolicy } from './eviction-policy';
import {
  ActionKey,
  DurableEvictionCache,
  DurableRunRecord,
  EvictionCause,
  buildEvictionReason,
} from './eviction-cache';
import { getErrorMessage } from '@hatchet/util/errors/hatchet-error';

export interface DurableEvictionConfig {
  /** How often we try selecting an eviction candidate. Default: 1000ms */
  checkIntervalMs?: number;
  /** How many slots to reserve from capacity-based eviction decisions. Default: 0 */
  reserveSlots?: number;
  /** Avoid immediately evicting runs that just entered a wait. Default: 10000ms */
  minWaitForCapacityEvictionMs?: number;
  /**
   * How long a shutdown waits for in-flight durable registrations (a waitFor or
   * runChildren whose engine ack has not arrived) to settle before giving up on
   * them. A run whose registration never settles is cancelled locally without
   * a report and left for the engine to reassign, like a run on a worker that
   * crashed. Default: 10000ms
   */
  shutdownRegistrationTimeoutMs?: number;
}

export const DEFAULT_DURABLE_EVICTION_CONFIG: Required<DurableEvictionConfig> = {
  checkIntervalMs: 1000,
  reserveSlots: 0,
  minWaitForCapacityEvictionMs: 10_000,
  shutdownRegistrationTimeoutMs: 10_000,
};

export class DurableEvictionManager {
  private _durableSlots: number;
  private _cancelLocal: (key: ActionKey, invocationCount: number) => void;
  private _requestEvictionWithAck: (key: ActionKey, rec: DurableRunRecord) => Promise<void>;
  private _config: Required<DurableEvictionConfig>;
  private _cache: DurableEvictionCache;
  private _logger: Logger;

  private _timer: ReturnType<typeof setInterval> | undefined;
  private _ticking = false;
  private _registrationWaiters = new Set<() => void>();

  constructor(opts: {
    durableSlots: number;
    cancelLocal: (key: ActionKey, invocationCount: number) => void;
    requestEvictionWithAck: (key: ActionKey, rec: DurableRunRecord) => Promise<void>;
    config?: DurableEvictionConfig;
    cache?: DurableEvictionCache;
    logger: Logger;
  }) {
    this._durableSlots = opts.durableSlots;
    this._cancelLocal = opts.cancelLocal;
    this._requestEvictionWithAck = opts.requestEvictionWithAck;
    this._config = { ...DEFAULT_DURABLE_EVICTION_CONFIG, ...opts.config };
    this._cache = opts.cache || new DurableEvictionCache();
    this._logger = opts.logger;
  }

  get cache(): DurableEvictionCache {
    return this._cache;
  }

  start(): void {
    if (this._timer) return;
    this._timer = setInterval(() => this._tickSafe(), this._config.checkIntervalMs);
  }

  stop(): void {
    if (this._timer) {
      clearInterval(this._timer);
      this._timer = undefined;
    }
  }

  registerRun(
    key: ActionKey,
    taskRunExternalId: string,
    invocationCount: number,
    evictionPolicy: EvictionPolicy | undefined
  ): void {
    this._cache.registerRun(key, taskRunExternalId, invocationCount, Date.now(), evictionPolicy);
  }

  unregisterRun(key: ActionKey): void {
    this._cache.unregisterRun(key);
    this._notifyRegistrationChange();
  }

  // A durable registration is in flight for this run: its waitFor or
  // runChildren has been sent and the engine ack is pending. markWaiting
  // consumes the registration; markRegistered ends one that failed.
  markRegistering(key: ActionKey, waitKind: string): void {
    this._cache.markRegistering(key, waitKind);
  }

  markRegistered(key: ActionKey): void {
    this._cache.markRegistered(key);
    this._notifyRegistrationChange();
  }

  markWaiting(
    key: ActionKey,
    waitKind: string,
    resourceId: string,
    consumesRegistration = false
  ): void {
    this._cache.markWaiting(key, Date.now(), waitKind, resourceId, consumesRegistration);
    this._notifyRegistrationChange();
  }

  private _notifyRegistrationChange(): void {
    for (const wake of this._registrationWaiters) {
      wake();
    }
  }

  // Waits, up to the configured bound, until no run has a registration in
  // flight. The engine only has a wait to evict once it has acked the
  // registration, and the ack must land before the listener stops, otherwise
  // the pending ack is rejected and the run fails instead of being evicted.
  private async _settleRegistrations(): Promise<void> {
    const timeoutMs = this._config.shutdownRegistrationTimeoutMs;
    const deadline = Date.now() + timeoutMs;

    while (this._cache.getAllRegistering().length > 0) {
      const remainingMs = deadline - Date.now();
      if (remainingMs <= 0) {
        return;
      }

      await new Promise<void>((resolve) => {
        const wake = () => {
          clearTimeout(timer);
          this._registrationWaiters.delete(wake);
          resolve();
        };
        const timer = setTimeout(wake, remainingMs);
        this._registrationWaiters.add(wake);
      });
    }
  }

  markActive(key: ActionKey): void {
    this._cache.markActive(key);
  }

  // The invocation count pins the teardown to the invocation that was
  // actually evicted: between requesting eviction and the server's ack, the
  // server may restore and re-dispatch the next invocation, which registers
  // under the same action key and must not be torn down.
  private _evictRun(key: ActionKey, invocationCount: number): void {
    const current = this._cache.get(key);
    if (current && current.invocationCount !== invocationCount) return;
    this._cancelLocal(key, invocationCount);
    this.unregisterRun(key);
  }

  private async _tickSafe(): Promise<void> {
    if (this._ticking) return;
    this._ticking = true;
    try {
      await this._tick();
    } catch (err: unknown) {
      this._logger.error(`DurableEvictionManager: error in eviction loop: ${getErrorMessage(err)}`);
    } finally {
      this._ticking = false;
    }
  }

  private async _tick(): Promise<void> {
    const evictedThisTick = new Set<ActionKey>();

    while (true) {
      const key = this._cache.selectEvictionCandidate(
        Date.now(),
        this._durableSlots,
        this._config.reserveSlots,
        this._config.minWaitForCapacityEvictionMs
      );

      if (!key) return;
      if (evictedThisTick.has(key)) return;
      evictedThisTick.add(key);

      const rec = this._cache.get(key);
      if (!rec || !rec.evictionPolicy) continue;

      this._logger.debug(
        `DurableEvictionManager: evicting task_run_external_id=${rec.taskRunExternalId} ` +
          `wait_kind=${rec.waitKind} resource_id=${rec.waitResourceId}`
      );

      await this._requestEvictionWithAck(key, rec);
      this._evictRun(key, rec.invocationCount);
    }
  }

  handleServerEviction(taskRunExternalId: string, invocationCount: number): void {
    const key = this._cache.findKeyByTaskRunExternalId(taskRunExternalId);
    if (!key) return;

    const rec = this._cache.get(key);
    if (rec && rec.invocationCount !== invocationCount) return;

    this._logger.info(
      `DurableEvictionManager: server-initiated eviction for task_run_external_id=${taskRunExternalId} invocation_count=${invocationCount}`
    );
    this._evictRun(key, invocationCount);
  }

  // A run with a registration that never settled must not be evicted, even if
  // it also has an acknowledged wait: the engine commits a wait's log entry
  // and its wake-up in separate transactions, an eviction can land between
  // them and the listener stop then cancels the second, and the acknowledged
  // wait may complete before the eviction commits, leaving an evicted run with
  // an unsatisfied entry that nothing resumes. Cancel it locally without a
  // report instead, exactly as if this worker had crashed: the engine
  // reassigns it once the heartbeat stops, and the replayed registration
  // re-triggers any entry left without a wake-up.
  private _leaveForReassignment(rec: DurableRunRecord): void {
    this._logger.warn(
      `DurableEvictionManager: durable registration for task_run_external_id=${rec.taskRunExternalId} ` +
        `still pending after ${this._config.shutdownRegistrationTimeoutMs}ms; ` +
        `leaving the run for reassignment instead of evicting it`
    );
    this._evictRun(rec.key, rec.invocationCount);
  }

  async evictAllWaiting(): Promise<number> {
    this.stop();

    await this._settleRegistrations();

    for (const rec of this._cache.getAllRegistering()) {
      this._leaveForReassignment(rec);
    }

    const waiting = this._cache.getAllWaiting();
    let evicted = 0;

    for (const rec of waiting) {
      // Earlier eviction acks were awaited, so this run may have moved on
      // since the snapshot: finished, been replaced by a newer invocation, or
      // completed its wait and started another registration.
      const current = this._cache.get(rec.key);
      if (!current || current.invocationCount !== rec.invocationCount) continue;
      if (current._registeringCount > 0) {
        this._leaveForReassignment(current);
        continue;
      }

      rec.evictionReason = buildEvictionReason(EvictionCause.WORKER_SHUTDOWN, rec);

      this._logger.debug(
        `DurableEvictionManager: shutdown-evicting task_run_external_id=${rec.taskRunExternalId} ` +
          `wait_kind=${rec.waitKind}`
      );

      try {
        await this._requestEvictionWithAck(rec.key, rec);
      } catch (err: unknown) {
        this._logger.error(
          `DurableEvictionManager: failed to send eviction for ` +
            `task_run_external_id=${rec.taskRunExternalId}: ${getErrorMessage(err)}`
        );
      }

      // Always cancel locally even if the server ACK failed, so the
      // future settles and exitGracefully doesn't hang.
      // This will get resolved by the reassignment of the task.
      this._evictRun(rec.key, rec.invocationCount);
      evicted++;
    }

    return evicted;
  }
}
