import { makeE2EClient, makeTestScope } from '../__e2e__/harness';
import { formatPublishLatency, measurePublishLatency } from './publish-latency';

// a benchmark, run only when asked for (test_durable_streams.sh --bench)
const describeBenchmark = process.env.HATCHET_E2E_BENCHMARKS ? describe : describe.skip;

describeBenchmark('durable-streams-e2e publish latency', () => {
  const hatchet = makeE2EClient();

  it('measures the latency of successive publishes to one topic', async () => {
    const measuredCount = 200;

    const stats = await measurePublishLatency(
      hatchet,
      makeTestScope('durable_streams_publish_latency'),
      'message',
      measuredCount
    );

    console.log(
      `durable streams publish latency over ${measuredCount} successive publishes: ${formatPublishLatency(stats)}`
    );

    expect(stats.count).toBe(measuredCount);

    // guards against a pathological regression, not a performance SLA
    expect(stats.avgMs).toBeLessThan(1_000);
  }, 180_000);
});
