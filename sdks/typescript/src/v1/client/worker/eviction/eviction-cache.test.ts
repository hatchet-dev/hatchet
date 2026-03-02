import {
  DurableEvictionCache,
  DurableRunRecord,
  EvictionCause,
  buildEvictionReason,
} from './eviction-cache';
import { EvictionPolicy } from './eviction-policy';

function makePolicy(overrides: Partial<EvictionPolicy> = {}): EvictionPolicy {
  return { ttl: undefined, allowCapacityEviction: true, priority: 0, ...overrides };
}

const T0 = 1_000_000;
const ONE_SEC = 1_000;
const ONE_MIN = 60_000;

describe('DurableEvictionCache', () => {
  let cache: DurableEvictionCache;

  beforeEach(() => {
    cache = new DurableEvictionCache();
  });

  // ------- basic bookkeeping -------

  describe('registerRun / unregisterRun / get', () => {
    it('registers and retrieves a run', () => {
      cache.registerRun('k1', 'ext-1', T0, makePolicy());
      const rec = cache.get('k1');
      expect(rec).toBeDefined();
      expect(rec!.taskRunExternalId).toBe('ext-1');
      expect(rec!.registeredAt).toBe(T0);
      expect(rec!.waitingSince).toBeUndefined();
    });

    it('returns undefined for unknown key', () => {
      expect(cache.get('nope')).toBeUndefined();
    });

    it('unregisters a run', () => {
      cache.registerRun('k1', 'ext-1', T0, makePolicy());
      cache.unregisterRun('k1');
      expect(cache.get('k1')).toBeUndefined();
    });
  });

  // ------- waiting state -------

  describe('markWaiting / markActive / getAllWaiting', () => {
    it('markWaiting sets wait fields', () => {
      cache.registerRun('k1', 'ext-1', T0, makePolicy());
      cache.markWaiting('k1', T0 + ONE_SEC, 'sleep', 'res-1');
      const rec = cache.get('k1')!;
      expect(rec.waitingSince).toBe(T0 + ONE_SEC);
      expect(rec.waitKind).toBe('sleep');
      expect(rec.waitResourceId).toBe('res-1');
    });

    it('markActive clears wait fields', () => {
      cache.registerRun('k1', 'ext-1', T0, makePolicy());
      cache.markWaiting('k1', T0 + ONE_SEC, 'sleep', 'res-1');
      cache.markActive('k1');
      const rec = cache.get('k1')!;
      expect(rec.waitingSince).toBeUndefined();
      expect(rec.waitKind).toBeUndefined();
      expect(rec.waitResourceId).toBeUndefined();
    });

    it('getAllWaiting returns only waiting runs', () => {
      cache.registerRun('k1', 'ext-1', T0, makePolicy());
      cache.registerRun('k2', 'ext-2', T0, makePolicy());
      cache.markWaiting('k1', T0, 'sleep', 'r1');
      expect(cache.getAllWaiting()).toHaveLength(1);
      expect(cache.getAllWaiting()[0].key).toBe('k1');
    });

    it('markWaiting on unknown key is a no-op', () => {
      cache.markWaiting('unknown', T0, 'sleep', 'r');
      expect(cache.get('unknown')).toBeUndefined();
    });

    it('markActive on unknown key is a no-op', () => {
      expect(() => cache.markActive('unknown')).not.toThrow();
    });
  });

  // ------- selectEvictionCandidate -------

  describe('selectEvictionCandidate', () => {
    const DURABLE_SLOTS = 4;
    const RESERVE_SLOTS = 0;
    const MIN_WAIT_MS = 5_000;

    it('returns undefined when no runs are registered', () => {
      expect(
        cache.selectEvictionCandidate(T0, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
      ).toBeUndefined();
    });

    it('returns undefined when runs exist but none are waiting', () => {
      cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: '1m' }));
      expect(
        cache.selectEvictionCandidate(T0 + ONE_MIN * 5, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
      ).toBeUndefined();
    });

    it('returns undefined when waiting runs have no eviction policy', () => {
      cache.registerRun('k1', 'ext-1', T0, undefined);
      cache.markWaiting('k1', T0, 'sleep', 'r');
      expect(
        cache.selectEvictionCandidate(T0 + ONE_MIN * 5, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
      ).toBeUndefined();
    });

    // ------- TTL eviction -------

    describe('TTL-based eviction', () => {
      it('evicts a run whose TTL has been exceeded', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: '1m' }));
        cache.markWaiting('k1', T0, 'sleep', 'r');
        const result = cache.selectEvictionCandidate(
          T0 + ONE_MIN + 1,
          DURABLE_SLOTS,
          RESERVE_SLOTS,
          MIN_WAIT_MS
        );
        expect(result).toBe('k1');
      });

      it('does not evict when TTL has not been exceeded', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: '5m' }));
        cache.markWaiting('k1', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + ONE_MIN, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
        ).toBeUndefined();
      });

      it('evicts regardless of capacity when TTL is exceeded', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: '1s' }));
        cache.markWaiting('k1', T0, 'sleep', 'r');
        const noCapacityPressureSlots = 100;
        expect(
          cache.selectEvictionCandidate(T0 + 2 * ONE_SEC, noCapacityPressureSlots, 0, MIN_WAIT_MS)
        ).toBe('k1');
      });

      it('picks lowest priority among TTL-eligible candidates', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: '1s', priority: 5 }));
        cache.registerRun('k2', 'ext-2', T0, makePolicy({ ttl: '1s', priority: 1 }));
        cache.markWaiting('k1', T0, 'sleep', 'r');
        cache.markWaiting('k2', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + 2 * ONE_SEC, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
        ).toBe('k2');
      });

      it('breaks priority ties by longest waiting (earliest waitingSince)', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: '1s', priority: 0 }));
        cache.registerRun('k2', 'ext-2', T0, makePolicy({ ttl: '1s', priority: 0 }));
        cache.markWaiting('k1', T0 + 100, 'sleep', 'r');
        cache.markWaiting('k2', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + 2 * ONE_SEC, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
        ).toBe('k2');
      });

      it('uses DurationObject TTL', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: { seconds: 30 } }));
        cache.markWaiting('k1', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + 29_000, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
        ).toBeUndefined();
        expect(
          cache.selectEvictionCandidate(T0 + 31_000, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS)
        ).toBe('k1');
      });
    });

    // ------- Capacity-based eviction -------

    describe('capacity-based eviction', () => {
      it('evicts under capacity pressure when min wait is met', () => {
        for (let i = 0; i < DURABLE_SLOTS; i += 1) {
          cache.registerRun(`k${i}`, `ext-${i}`, T0, makePolicy());
          cache.markWaiting(`k${i}`, T0, 'sleep', 'r');
        }
        const result = cache.selectEvictionCandidate(
          T0 + MIN_WAIT_MS + 1,
          DURABLE_SLOTS,
          RESERVE_SLOTS,
          MIN_WAIT_MS
        );
        expect(result).toBeDefined();
      });

      it('does not evict when there is no capacity pressure', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy());
        cache.markWaiting('k1', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(
            T0 + MIN_WAIT_MS + 1,
            DURABLE_SLOTS,
            RESERVE_SLOTS,
            MIN_WAIT_MS
          )
        ).toBeUndefined();
      });

      it('does not evict when min wait threshold has not been met', () => {
        for (let i = 0; i < DURABLE_SLOTS; i += 1) {
          cache.registerRun(`k${i}`, `ext-${i}`, T0, makePolicy());
          cache.markWaiting(`k${i}`, T0, 'sleep', 'r');
        }
        expect(
          cache.selectEvictionCandidate(
            T0 + MIN_WAIT_MS - 1,
            DURABLE_SLOTS,
            RESERVE_SLOTS,
            MIN_WAIT_MS
          )
        ).toBeUndefined();
      });

      it('respects allowCapacityEviction=false', () => {
        for (let i = 0; i < DURABLE_SLOTS; i += 1) {
          cache.registerRun(`k${i}`, `ext-${i}`, T0, makePolicy({ allowCapacityEviction: false }));
          cache.markWaiting(`k${i}`, T0, 'sleep', 'r');
        }
        expect(
          cache.selectEvictionCandidate(
            T0 + MIN_WAIT_MS + 1,
            DURABLE_SLOTS,
            RESERVE_SLOTS,
            MIN_WAIT_MS
          )
        ).toBeUndefined();
      });

      it('skips allowCapacityEviction=false but evicts others', () => {
        cache.registerRun('protected', 'ext-p', T0, makePolicy({ allowCapacityEviction: false }));
        cache.markWaiting('protected', T0, 'sleep', 'r');
        for (let i = 1; i < DURABLE_SLOTS; i += 1) {
          cache.registerRun(`k${i}`, `ext-${i}`, T0, makePolicy());
          cache.markWaiting(`k${i}`, T0, 'sleep', 'r');
        }
        const result = cache.selectEvictionCandidate(
          T0 + MIN_WAIT_MS + 1,
          DURABLE_SLOTS,
          RESERVE_SLOTS,
          MIN_WAIT_MS
        );
        expect(result).toBeDefined();
        expect(result).not.toBe('protected');
      });

      it('picks lowest priority among capacity candidates', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ priority: 10 }));
        cache.registerRun('k2', 'ext-2', T0, makePolicy({ priority: 2 }));
        cache.registerRun('k3', 'ext-3', T0, makePolicy({ priority: 5 }));
        cache.registerRun('k4', 'ext-4', T0, makePolicy({ priority: 7 }));
        for (const k of ['k1', 'k2', 'k3', 'k4']) {
          cache.markWaiting(k, T0, 'sleep', 'r');
        }
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1, 4, RESERVE_SLOTS, MIN_WAIT_MS)
        ).toBe('k2');
      });

      it('breaks capacity priority ties by longest waiting', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ priority: 0 }));
        cache.registerRun('k2', 'ext-2', T0, makePolicy({ priority: 0 }));
        cache.registerRun('k3', 'ext-3', T0, makePolicy({ priority: 0 }));
        cache.registerRun('k4', 'ext-4', T0, makePolicy({ priority: 0 }));
        cache.markWaiting('k1', T0 + 200, 'sleep', 'r');
        cache.markWaiting('k2', T0, 'sleep', 'r');
        cache.markWaiting('k3', T0 + 100, 'sleep', 'r');
        cache.markWaiting('k4', T0 + 300, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1000, 4, RESERVE_SLOTS, MIN_WAIT_MS)
        ).toBe('k2');
      });
    });

    // ------- reserveSlots -------

    describe('reserveSlots', () => {
      it('reserve slots reduce the effective capacity threshold', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy());
        cache.registerRun('k2', 'ext-2', T0, makePolicy());
        cache.registerRun('k3', 'ext-3', T0, makePolicy());
        for (const k of ['k1', 'k2', 'k3']) {
          cache.markWaiting(k, T0, 'sleep', 'r');
        }

        // 4 slots - 2 reserved = 2 effective; 3 waiting >= 2 → pressure
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1, 4, 2, MIN_WAIT_MS)
        ).toBeDefined();
      });

      it('no pressure when waiting count is below effective threshold', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy());
        cache.markWaiting('k1', T0, 'sleep', 'r');

        // 4 slots - 0 reserved = 4 effective; 1 waiting < 4 → no pressure
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1, 4, 0, MIN_WAIT_MS)
        ).toBeUndefined();
      });

      it('reserveSlots >= durableSlots means no capacity eviction', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy());
        cache.markWaiting('k1', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1, 4, 4, MIN_WAIT_MS)
        ).toBeUndefined();
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1, 4, 5, MIN_WAIT_MS)
        ).toBeUndefined();
      });
    });

    // ------- durableSlots edge cases -------

    describe('durableSlots edge cases', () => {
      it('durableSlots=0 means no capacity eviction', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy());
        cache.markWaiting('k1', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1, 0, 0, MIN_WAIT_MS)
        ).toBeUndefined();
      });

      it('durableSlots negative means no capacity eviction', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy());
        cache.markWaiting('k1', T0, 'sleep', 'r');
        expect(
          cache.selectEvictionCandidate(T0 + MIN_WAIT_MS + 1, -1, 0, MIN_WAIT_MS)
        ).toBeUndefined();
      });
    });

    // ------- TTL priority over capacity -------

    describe('TTL takes precedence over capacity', () => {
      it('selects TTL-eligible candidate even when capacity candidates also exist', () => {
        cache.registerRun('ttl-run', 'ext-t', T0, makePolicy({ ttl: '1s', priority: 10 }));
        cache.registerRun('cap-run', 'ext-c', T0, makePolicy({ priority: 0 }));
        cache.markWaiting('ttl-run', T0, 'sleep', 'r');
        cache.markWaiting('cap-run', T0, 'sleep', 'r');

        const result = cache.selectEvictionCandidate(T0 + 2 * ONE_SEC, 2, 0, MIN_WAIT_MS);
        expect(result).toBe('ttl-run');
      });
    });

    // ------- evictionReason side-effect -------

    describe('evictionReason is set on the chosen record', () => {
      it('sets TTL reason on TTL eviction', () => {
        cache.registerRun('k1', 'ext-1', T0, makePolicy({ ttl: '30s' }));
        cache.markWaiting('k1', T0, 'sleep', 'res-1');
        cache.selectEvictionCandidate(T0 + 31_000, DURABLE_SLOTS, RESERVE_SLOTS, MIN_WAIT_MS);
        expect(cache.get('k1')!.evictionReason).toMatch(/TTL.*exceeded/);
      });

      it('sets capacity reason on capacity eviction', () => {
        for (let i = 0; i < DURABLE_SLOTS; i += 1) {
          cache.registerRun(`k${i}`, `ext-${i}`, T0, makePolicy());
          cache.markWaiting(`k${i}`, T0, 'sleep', 'res');
        }
        const key = cache.selectEvictionCandidate(
          T0 + MIN_WAIT_MS + 1,
          DURABLE_SLOTS,
          RESERVE_SLOTS,
          MIN_WAIT_MS
        )!;
        expect(cache.get(key)!.evictionReason).toMatch(/capacity/i);
      });
    });
  });
});

describe('buildEvictionReason', () => {
  function makeRecord(overrides: Partial<DurableRunRecord> = {}): DurableRunRecord {
    return {
      key: 'k1',
      taskRunExternalId: 'ext-1',
      evictionPolicy: { ttl: '30s', allowCapacityEviction: true, priority: 0 },
      registeredAt: T0,
      waitingSince: T0,
      waitKind: 'sleep',
      waitResourceId: 'res-1',
      evictionReason: undefined,
      ...overrides,
    };
  }

  it('formats TTL_EXCEEDED with ttl and resource', () => {
    const reason = buildEvictionReason(EvictionCause.TTL_EXCEEDED, makeRecord());
    expect(reason).toBe('Wait TTL (30s) exceeded while waiting on sleep(res-1)');
  });

  it('formats TTL_EXCEEDED without ttl', () => {
    const reason = buildEvictionReason(
      EvictionCause.TTL_EXCEEDED,
      makeRecord({ evictionPolicy: { allowCapacityEviction: true } })
    );
    expect(reason).toBe('Wait TTL exceeded while waiting on sleep(res-1)');
  });

  it('formats CAPACITY_PRESSURE', () => {
    const reason = buildEvictionReason(EvictionCause.CAPACITY_PRESSURE, makeRecord());
    expect(reason).toBe('Worker at capacity while waiting on sleep(res-1)');
  });

  it('formats WORKER_SHUTDOWN', () => {
    const reason = buildEvictionReason(EvictionCause.WORKER_SHUTDOWN, makeRecord());
    expect(reason).toBe('Worker shutdown while waiting on sleep(res-1)');
  });

  it('handles missing waitKind', () => {
    const reason = buildEvictionReason(
      EvictionCause.CAPACITY_PRESSURE,
      makeRecord({ waitKind: undefined, waitResourceId: undefined })
    );
    expect(reason).toBe('Worker at capacity while waiting on unknown');
  });

  it('handles waitKind without resourceId', () => {
    const reason = buildEvictionReason(
      EvictionCause.TTL_EXCEEDED,
      makeRecord({ waitResourceId: undefined })
    );
    expect(reason).toBe('Wait TTL (30s) exceeded while waiting on sleep');
  });
});
