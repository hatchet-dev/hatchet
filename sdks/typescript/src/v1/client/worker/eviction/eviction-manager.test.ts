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

      expect(cancelLocal).toHaveBeenCalledWith('run-1/0');
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
      expect(cancelLocal).toHaveBeenCalledWith('run-1/0');
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

      expect(cancelLocal).toHaveBeenCalledWith('run-1/0');
      expect(manager.cache.get('run-1/0')).toBeUndefined();
    });
  });
});
