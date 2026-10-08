import { randomBytes } from 'crypto';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';
import {
  describeBenchmark,
  formatPublishLatency,
  formatSize,
  maxInlinePayloadBytes,
  measurePublishLatency,
  MiB,
  PublishLatencyStats,
} from './benchmark';

const payloadSizes = [16, 1024, 64 * 1024, 512 * 1024, MiB, 2 * MiB, maxInlinePayloadBytes];

describeBenchmark('durable-streams-e2e publish latency by payload size', () => {
  const hatchet = makeE2EClient();

  it('measures the latency of successive publishes at each payload size', async () => {
    const measuredCount = 50;
    const results: { size: number; stats: PublishLatencyStats }[] = [];

    for (const size of payloadSizes) {
      // random bytes, so payload compression can't hide the cost of large messages
      const payload = new Uint8Array(randomBytes(size));
      const stats = await measurePublishLatency(
        hatchet,
        makeTestScope(`durable_streams_publish_latency_${size}`),
        payload,
        measuredCount
      );

      results.push({ size, stats });
    }

    console.log(
      [
        `durable streams publish latency by payload size, ${measuredCount} successive publishes each:`,
        ...results.map(
          ({ size, stats }) => `  ${formatSize(size).padStart(8)}  ${formatPublishLatency(stats)}`
        ),
      ].join('\n')
    );

    for (const { stats } of results) {
      expect(stats.count).toBe(measuredCount);

      // guards against a pathological regression, not a performance SLA
      expect(stats.avgMs).toBeLessThan(2_000);
    }
  }, 300_000);
});
