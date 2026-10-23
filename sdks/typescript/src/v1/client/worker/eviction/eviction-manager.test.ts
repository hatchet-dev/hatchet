import { Logger } from '@hatchet/util/logger';
import { DurableEvictionManager } from './eviction-manager';
import { DurableEvictionCache } from './eviction-cache';

class NoopLogger extends Logger {
  debug() {}
  info() {}
  green() {}
  warn() {}
  error() {}
  util() {}
}

function makeManager() {
  const cancelLocal = jest.fn();
  const requestEvictionWithAck = jest.fn().mockResolvedValue(undefined);
  const cache = new DurableEvictionCache();

  const manager = new DurableEvictionManager({
    durableSlots: 10,
    cancelLocal,
    requestEvictionWithAck,
    config: { checkIntervalMs: 3_600_000 },
    cache,
    logger: new NoopLogger(),
  });

  return { manager, cancelLocal, requestEvictionWithAck, cache };
}

describe('DurableEvictionManager', () => {
  describe('handleServerEviction', () => {
    it('cancels and unregisters the matching run when invocationCount matches', () => {
      const { manager, cancelLocal } = makeManager();

      manager.registerRun('run-1/0', 'ext-1', 2, {
        ttl: '30s',
        allowCapacityEviction: true,
        priority: 0,
      });
      manager.markWaiting('run-1/0', 'sleep', 's1');

      manager.handleServerEviction('ext-1', 2);

      expect(cancelLocal).toHaveBeenCalledWith('run-1/0', 2);
      expect(manager.cache.get('run-1/0')).toBeUndefined();
    });

    it('is a no-op for unknown taskRunExternalId', () => {
      const { manager, cancelLocal } = makeManager();

      manager.registerRun('run-1/0', 'ext-1', 1, undefined);

      manager.handleServerEviction('no-such-id', 1);

      expect(cancelLocal).not.toHaveBeenCalled();
      expect(manager.cache.get('run-1/0')).toBeDefined();
    });

    it('only evicts the matching run, not others', () => {
      const { manager, cancelLocal } = makeManager();

      manager.registerRun('run-1/0', 'ext-1', 1, {
        ttl: '30s',
        allowCapacityEviction: true,
        priority: 0,
      });
      manager.registerRun('run-2/0', 'ext-2', 1, {
        ttl: '30s',
        allowCapacityEviction: true,
        priority: 0,
      });
      manager.markWaiting('run-1/0', 'sleep', 's1');
      manager.markWaiting('run-2/0', 'sleep', 's2');

      manager.handleServerEviction('ext-1', 1);

      expect(cancelLocal).toHaveBeenCalledTimes(1);
      expect(cancelLocal).toHaveBeenCalledWith('run-1/0', 1);
      expect(manager.cache.get('run-1/0')).toBeUndefined();
      expect(manager.cache.get('run-2/0')).toBeDefined();
    });

    it('does not evict when invocationCount does not match (newer invocation)', () => {
      const { manager, cancelLocal } = makeManager();

      manager.registerRun('run-1/0', 'ext-1', 3, {
        ttl: '30s',
        allowCapacityEviction: true,
        priority: 0,
      });
      manager.markWaiting('run-1/0', 'sleep', 's1');

      manager.handleServerEviction('ext-1', 2);

      expect(cancelLocal).not.toHaveBeenCalled();
      expect(manager.cache.get('run-1/0')).toBeDefined();
    });

    it('evicts when invocationCount matches exactly', () => {
      const { manager, cancelLocal } = makeManager();

      manager.registerRun('run-1/0', 'ext-1', 5, {
        ttl: '30s',
        allowCapacityEviction: true,
        priority: 0,
      });
      manager.markWaiting('run-1/0', 'sleep', 's1');

      manager.handleServerEviction('ext-1', 5);

      expect(cancelLocal).toHaveBeenCalledWith('run-1/0', 5);
      expect(manager.cache.get('run-1/0')).toBeUndefined();
    });
  });

  describe('evictAllWaiting shutdown drain', () => {
    it('evicts a run that enters a wait after the drain snapshot', async () => {
      const { manager, cancelLocal, requestEvictionWithAck } = makeManager();

      manager.registerRun('run-1/0', 'ext-1', 1, {
        ttl: '30s',
        allowCapacityEviction: true,
        priority: 0,
      });

      // The run is not waiting yet when shutdown starts, so the snapshot
      // misses it; markWaiting afterwards must evict it rather than leave it
      // to be rejected when the durable listener stops.
      const evicted = await manager.evictAllWaiting();
      expect(evicted).toBe(0);

      manager.markWaiting('run-1/0', 'waitFor', 'registering');

      // The eviction ack round-trip is async; the local cancel always lands.
      await new Promise(process.nextTick);
      await Promise.resolve();

      expect(requestEvictionWithAck).toHaveBeenCalledWith(
        'run-1/0',
        expect.objectContaining({ taskRunExternalId: 'ext-1', invocationCount: 1 })
      );
      expect(cancelLocal).toHaveBeenCalledWith('run-1/0', 1);
      expect(manager.cache.get('run-1/0')).toBeUndefined();
    });

    it('does not re-evict a key already being drained for shutdown', async () => {
      const { manager, requestEvictionWithAck } = makeManager();
      let resolveAck!: () => void;
      requestEvictionWithAck.mockReturnValue(
        new Promise<void>((resolve) => {
          resolveAck = resolve;
        })
      );

      manager.registerRun('run-1/0', 'ext-1', 1, {
        ttl: '30s',
        allowCapacityEviction: true,
        priority: 0,
      });
      await manager.evictAllWaiting();

      manager.markWaiting('run-1/0', 'waitFor', 'registering');
      manager.markWaiting('run-1/0', 'waitFor', 'registering');

      expect(requestEvictionWithAck).toHaveBeenCalledTimes(1);
      resolveAck();
      await Promise.resolve();
    });
  });
});
