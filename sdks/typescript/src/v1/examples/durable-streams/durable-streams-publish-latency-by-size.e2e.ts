import { randomBytes } from 'crypto';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';
import {
  formatPublishLatency,
  measurePublishLatency,
  PublishLatencyStats,
} from './publish-latency';

// the server's per-message limit (MaxStreamMessagePayloadBytes), just under gRPC's 4 MiB
const maxPayloadBytes = 4 * 1024 * 1024 - 5 * 1024;

const payloadSizes = [
  16,
  1024,
  64 * 1024,
  512 * 1024,
  1024 * 1024,
  2 * 1024 * 1024,
  maxPayloadBytes,
];

function formatSize(bytes: number): string {
  if (bytes % (1024 * 1024) === 0) return `${bytes / (1024 * 1024)}MiB`;
  return bytes >= 1024 ? `${bytes / 1024}KiB` : `${bytes}B`;
}

// a benchmark, run only when asked for (test_durable_streams.sh --bench)
const describeBenchmark = process.env.HATCHET_E2E_BENCHMARKS ? describe : describe.skip;

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
