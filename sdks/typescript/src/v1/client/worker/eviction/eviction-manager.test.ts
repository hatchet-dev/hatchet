import { Logger } from '@hatchet/util/logger';
import { DurableEvictionConfig, DurableEvictionManager } from './eviction-manager';
import { DurableEvictionCache } from './eviction-cache';

class NoopLogger extends Logger {
  debug() {}
  info() {}
  green() {}
  warn() {}
  error() {}
  util() {}
}

function makeManager(config: DurableEvictionConfig = {}) {
  const cancelLocal = jest.fn();
  const requestEvictionWithAck = jest.fn().mockResolvedValue(undefined);
  const cache = new DurableEvictionCache();

  const manager = new DurableEvictionManager({
    durableSlots: 10,
    cancelLocal,
    requestEvictionWithAck,
    config: { checkIntervalMs: 3_600_000, ...config },
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

  describe('evictAllWaiting', () => {
    beforeEach(() => {
      jest.useFakeTimers();
    });

    afterEach(() => {
      jest.useRealTimers();
    });

    it('waits for an in-flight registration to become a wait, then evicts it', async () => {
      const { manager, cancelLocal, requestEvictionWithAck } = makeManager();
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markRegistering('run-1/0', 'waitFor');

      const evicting = manager.evictAllWaiting();
      await jest.advanceTimersByTimeAsync(5_000);
      expect(requestEvictionWithAck).not.toHaveBeenCalled();

      manager.markWaiting('run-1/0', 'waitFor', 'sleep-1', true);
      await expect(evicting).resolves.toBe(1);

      expect(requestEvictionWithAck).toHaveBeenCalledTimes(1);
      expect(requestEvictionWithAck.mock.calls[0][1].evictionReason).toBe(
        'Worker shutdown while waiting on waitFor(sleep-1)'
      );
      expect(cancelLocal).toHaveBeenCalledWith('run-1/0', 1);
    });

    it('stops waiting when a registration fails, without evicting the run', async () => {
      const { manager, cancelLocal, requestEvictionWithAck } = makeManager();
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markRegistering('run-1/0', 'waitFor');

      const evicting = manager.evictAllWaiting();
      manager.markRegistered('run-1/0');

      await expect(evicting).resolves.toBe(0);
      expect(requestEvictionWithAck).not.toHaveBeenCalled();
      expect(cancelLocal).not.toHaveBeenCalled();
    });

    it('stops waiting when a registering run is unregistered', async () => {
      const { manager, requestEvictionWithAck } = makeManager();
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markRegistering('run-1/0', 'waitFor');

      const evicting = manager.evictAllWaiting();
      manager.unregisterRun('run-1/0');

      await expect(evicting).resolves.toBe(0);
      expect(requestEvictionWithAck).not.toHaveBeenCalled();
    });

    it('leaves a run whose registration never settles for reassignment: cancelled locally, not evicted', async () => {
      const { manager, cancelLocal, requestEvictionWithAck } = makeManager({
        shutdownRegistrationTimeoutMs: 500,
      });
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markRegistering('run-1/0', 'waitFor');

      const evicting = manager.evictAllWaiting();
      await jest.advanceTimersByTimeAsync(499);
      expect(cancelLocal).not.toHaveBeenCalled();

      await jest.advanceTimersByTimeAsync(1);
      await expect(evicting).resolves.toBe(0);

      expect(requestEvictionWithAck).not.toHaveBeenCalled();
      expect(cancelLocal).toHaveBeenCalledWith('run-1/0', 1);
      expect(manager.cache.get('run-1/0')).toBeUndefined();
    });

    it('also leaves a waiting run for reassignment while another of its registrations is pending', async () => {
      const { manager, cancelLocal, requestEvictionWithAck } = makeManager({
        shutdownRegistrationTimeoutMs: 500,
      });
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markWaiting('run-1/0', 'sleep', 's1');
      manager.markRegistering('run-1/0', 'waitFor');
      manager.registerRun('run-2/0', 'ext-2', 1, undefined);
      manager.markWaiting('run-2/0', 'sleep', 's2');

      const evicting = manager.evictAllWaiting();
      await jest.advanceTimersByTimeAsync(500);
      await expect(evicting).resolves.toBe(1);

      // Only the run whose waits are all acknowledged is evicted.
      expect(requestEvictionWithAck).toHaveBeenCalledTimes(1);
      expect(requestEvictionWithAck.mock.calls[0][1].taskRunExternalId).toBe('ext-2');
      expect(cancelLocal.mock.calls.map(([key]) => key)).toEqual(['run-1/0', 'run-2/0']);
    });

    it('rechecks a snapshotted run before its eviction: a registration started meanwhile leaves it for reassignment', async () => {
      const { manager, cancelLocal, requestEvictionWithAck } = makeManager();
      let releaseFirstAck = () => {};
      requestEvictionWithAck.mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            releaseFirstAck = resolve;
          })
      );
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markWaiting('run-1/0', 'sleep', 's1', true);
      manager.registerRun('run-2/0', 'ext-2', 1, undefined);
      manager.markWaiting('run-2/0', 'sleep', 's2', true);

      const evicting = manager.evictAllWaiting();
      await jest.advanceTimersByTimeAsync(0);
      expect(requestEvictionWithAck).toHaveBeenCalledTimes(1);

      // While the first eviction ack is pending, the second run's wait completes
      // and its task registers another wait that the engine has not acked.
      manager.markActive('run-2/0');
      manager.markRegistering('run-2/0', 'waitFor');

      releaseFirstAck();
      await expect(evicting).resolves.toBe(1);

      expect(requestEvictionWithAck).toHaveBeenCalledTimes(1);
      expect(requestEvictionWithAck.mock.calls[0][1].taskRunExternalId).toBe('ext-1');
      expect(cancelLocal.mock.calls.map(([key]) => key)).toEqual(['run-1/0', 'run-2/0']);
      expect(manager.cache.get('run-2/0')).toBeUndefined();
    });

    it('skips a snapshotted run that finished or was replaced before its eviction', async () => {
      const { manager, cancelLocal, requestEvictionWithAck } = makeManager();
      let releaseFirstAck = () => {};
      requestEvictionWithAck.mockImplementationOnce(
        () =>
          new Promise<void>((resolve) => {
            releaseFirstAck = resolve;
          })
      );
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markWaiting('run-1/0', 'sleep', 's1', true);
      manager.registerRun('run-2/0', 'ext-2', 1, undefined);
      manager.markWaiting('run-2/0', 'sleep', 's2', true);
      manager.registerRun('run-3/0', 'ext-3', 1, undefined);
      manager.markWaiting('run-3/0', 'sleep', 's3', true);

      const evicting = manager.evictAllWaiting();
      await jest.advanceTimersByTimeAsync(0);

      manager.unregisterRun('run-2/0');
      manager.registerRun('run-3/0', 'ext-3', 2, undefined);
      manager.markWaiting('run-3/0', 'sleep', 's3', true);

      releaseFirstAck();
      await expect(evicting).resolves.toBe(1);

      expect(requestEvictionWithAck.mock.calls.map(([, rec]) => rec.taskRunExternalId)).toEqual([
        'ext-1',
      ]);
      expect(cancelLocal.mock.calls.map(([key]) => key)).toEqual(['run-1/0']);
      expect(manager.cache.get('run-3/0')?.invocationCount).toBe(2);
    });

    it('evicts already-waiting runs without delay when nothing is registering', async () => {
      const { manager, requestEvictionWithAck } = makeManager();
      manager.registerRun('run-1/0', 'ext-1', 1, undefined);
      manager.markWaiting('run-1/0', 'sleep', 's1');

      await expect(manager.evictAllWaiting()).resolves.toBe(1);
      expect(requestEvictionWithAck).toHaveBeenCalledTimes(1);
    });
  });
});
