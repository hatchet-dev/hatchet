import { makeE2EClient, checkDurableEvictionSupport } from '../__e2e__/harness';
import { callbackOrderingRoot, ROOT_DEFAULTS } from './workflow';

describe('durable-callback-ordering-e2e', () => {
  const hatchet = makeE2EClient();
  let evictionSupported = false;

  beforeAll(async () => {
    evictionSupported = await checkDurableEvictionSupport(hatchet);
  });

  it('replayed completions resume in recorded order', async () => {
    if (!evictionSupported) {
      console.log('Skipping: engine does not support durable eviction');
      return;
    }

    const inputs = Array.from({ length: 25 }, () => ({}));

    let results;
    try {
      results = await callbackOrderingRoot.run(inputs);
    } catch (error) {
      if (String(error).includes('NonDeterminismError')) {
        throw new Error(
          `replayed completions were consumed out of recorded order:\n${String(error)}`,
          { cause: error }
        );
      }
      throw error;
    }

    for (const result of results) {
      expect([...result.completedMids].sort((a, b) => a - b)).toEqual(
        Array.from({ length: ROOT_DEFAULTS.durables }, (_, i) => i)
      );
      expect(result.midInvocationCounts).toHaveLength(ROOT_DEFAULTS.durables);
    }

    // The worker's TTL sweep evicts one waiting durable run per server round
    // trip, oldest wait first, so under load it does not reach every mid before
    // that mid completes. Which mids get replayed is not guaranteed; that at
    // least one does is, since the sweep starts on the oldest waiting mid within
    // a second and a mid idles for far longer than one round trip.
    const replayedMids = results
      .flatMap((result) => result.midInvocationCounts)
      .filter((count) => count >= 2).length;
    expect(replayedMids).toBeGreaterThan(0);
  }, 300_000);
});
