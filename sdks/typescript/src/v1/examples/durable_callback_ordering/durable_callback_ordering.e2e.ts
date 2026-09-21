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
      expect(Math.max(...result.midInvocationCounts)).toBeGreaterThanOrEqual(2);
    }
  }, 300_000);
});
