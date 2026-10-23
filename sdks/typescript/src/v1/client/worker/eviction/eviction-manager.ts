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
}

export const DEFAULT_DURABLE_EVICTION_CONFIG: Required<DurableEvictionConfig> = {
  checkIntervalMs: 1000,
  reserveSlots: 0,
  minWaitForCapacityEvictionMs: 10_000,
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

  // Set once shutdown eviction starts. While set, every newly marked wait is
  // evicted immediately so a run cannot register a durable wait past the
  // evictAllWaiting snapshot and then be killed by the listener stopping.
  private _shuttingDown = false;
  private _inflightShutdownEvictions = new Set<ActionKey>();

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
  }

  markWaiting(key: ActionKey, waitKind: string, resourceId: string): void {
    this._cache.markWaiting(key, Date.now(), waitKind, resourceId);
    if (this._shuttingDown) {
      this._evictForShutdown(key);
    }
  }

  updateWaiting(key: ActionKey, waitKind: string, resourceId: string): void {
    this._cache.updateWaiting(key, waitKind, resourceId);
  }

  markActive(key: ActionKey): void {
    this._cache.markActive(key);
  }

  private _evictForShutdown(key: ActionKey): void {
    if (this._inflightShutdownEvictions.has(key)) return;
    const rec = this._cache.get(key);
    if (!rec) return;

    this._inflightShutdownEvictions.add(key);
    rec.evictionReason = buildEvictionReason(EvictionCause.WORKER_SHUTDOWN, rec);

    this._requestEvictionWithAck(key, rec)
      .catch((err: unknown) => {
        this._logger.error(
          `DurableEvictionManager: failed to send eviction for ` +
            `task_run_external_id=${rec.taskRunExternalId}: ${getErrorMessage(err)}`
        );
      })
      .finally(() => {
        this._inflightShutdownEvictions.delete(key);
        // Always cancel locally even if the server ACK failed, so the future
        // settles and exitGracefully doesn't hang.
        this._evictRun(key, rec.invocationCount);
      });
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

  async evictAllWaiting(): Promise<number> {
    this.stop();
    // Flipping the drain flag before the snapshot means a run that enters a
    // wait while (or after) this drain runs is evicted on mark instead of
    // slipping past it and failing when the durable listener stops.
    this._shuttingDown = true;

    const waiting = this._cache.getAllWaiting();
    let evicted = 0;

    for (const rec of waiting) {
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
