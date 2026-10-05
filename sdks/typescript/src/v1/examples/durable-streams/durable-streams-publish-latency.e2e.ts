import { makeE2EClient, makeTestScope } from '../__e2e__/harness';
import { describeBenchmark, formatPublishLatency, measurePublishLatency } from './benchmark';

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
