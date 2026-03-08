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
  private _cancelLocal: (key: ActionKey) => void;
  private _requestEvictionWithAck: (key: ActionKey, rec: DurableRunRecord) => Promise<void>;
  private _config: Required<DurableEvictionConfig>;
  private _cache: DurableEvictionCache;
  private _logger: Logger;

  private _timer: ReturnType<typeof setInterval> | undefined;
  private _ticking = false;

  constructor(opts: {
    durableSlots: number;
    cancelLocal: (key: ActionKey) => void;
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
  }

  markActive(key: ActionKey): void {
    this._cache.markActive(key);
  }

  private _evictRun(key: ActionKey): void {
    this._cancelLocal(key);
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
      this._evictRun(key);
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
    this._evictRun(key);
  }

  async evictAllWaiting(): Promise<number> {
    this.stop();

    const waiting = this._cache.getAllWaiting();
    let evicted = 0;

    for (const rec of waiting) {
      if (!rec.evictionPolicy) continue;

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
        continue;
      }

      this._evictRun(rec.key);
      evicted++;
    }

    return evicted;
  }
}
