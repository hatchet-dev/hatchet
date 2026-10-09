import { randomBytes } from 'crypto';
import { performance } from 'perf_hooks';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';
import {
  describeBenchmark,
  formatLatency,
  formatSize,
  latencyStats,
  maxInlinePayloadBytes,
  MiB,
  openLiveSubscription,
} from './benchmark';

const cases = [
  { size: 1 * MiB, count: 10 },
  { size: maxInlinePayloadBytes, count: 10 },
  { size: 4 * MiB, count: 10 },
  { size: 8 * MiB, count: 10 },
  { size: 16 * MiB, count: 10 },
  { size: 32 * MiB, count: 5 },
  { size: 50 * MiB, count: 5 },
  { size: 64 * MiB, count: 5 },
];

type Sample = {
  publishMs: number;
  uploadMs?: number;
  fetchMs?: number;
  deliveredMs: number;
};

// filled in by the API wrappers for the message in flight
type HttpTrace = { uploadMs?: number; fetchMs?: number };

const summarize = (valuesMs: number[]) => formatLatency(latencyStats(valuesMs), 0);

describeBenchmark('durable-streams-e2e large payload latency', () => {
  const hatchet = makeE2EClient();
  const { api } = hatchet;
  const realUpload = api.v1StreamPayloadUpload;
  const realGet = api.v1StreamPayloadGet;
  let trace: HttpTrace = {};

  beforeAll(() => {
    // splits the upload and the subscriber's fetch out of the totals
    api.v1StreamPayloadUpload = async (...args) => {
      const start = performance.now();
      const res = await realUpload(...args);
      trace.uploadMs = performance.now() - start;
      return res;
    };
    api.v1StreamPayloadGet = async (...args) => {
      const start = performance.now();
      const res = await realGet(...args);
      trace.fetchMs = performance.now() - start;
      return res;
    };
  });

  afterAll(() => {
    api.v1StreamPayloadUpload = realUpload;
    api.v1StreamPayloadGet = realGet;
  });

  it('measures publish and delivery latency up to the large payload limit', async () => {
    const results: { size: number; samples: Sample[] }[] = [];

    for (const { size, count } of cases) {
      const topic = makeTestScope(`durable_streams_large_payload_${size}`);
      const subscription = await openLiveSubscription(hatchet, topic);

      // random bytes, so compression can't hide the cost of large messages
      const payload = new Uint8Array(randomBytes(size));
      const samples: Sample[] = [];

      // one unmeasured message absorbs topic and connection warmup at this size
      for (let i = 0; i < count + 1; i += 1) {
        trace = {};
        const delivered = subscription.nextEvent();

        const start = performance.now();
        await hatchet.streams.publish(topic, payload);
        const publishMs = performance.now() - start;
        const ev = await delivered;
        const deliveredMs = performance.now() - start;

        expect(Buffer.compare(Buffer.from(ev.payload), Buffer.from(payload))).toBe(0);

        if (i > 0) {
          samples.push({ publishMs, deliveredMs, ...trace });
        }
      }

      await subscription.close();

      results.push({ size, samples });
      console.log(`done ${formatSize(size)}`);
    }

    console.log(
      [
        'durable streams large payload latency (ms); delivered = publish start until the subscriber has the bytes:',
        ...results.map(({ size, samples }) => {
          const path = size > maxInlinePayloadBytes ? 'upload' : 'inline';
          const uploads = samples.flatMap((s) => (s.uploadMs === undefined ? [] : [s.uploadMs]));
          const fetches = samples.flatMap((s) => (s.fetchMs === undefined ? [] : [s.fetchMs]));
          const delivered = samples.map((s) => s.deliveredMs);
          const { avgMs: avgDelivered } = latencyStats(delivered);

          return [
            `  ${formatSize(size).padStart(9)} ${path.padEnd(6)} n=${samples.length}`,
            `publish: ${summarize(samples.map((s) => s.publishMs))}`,
            uploads.length ? `upload: ${summarize(uploads)}` : '',
            fetches.length ? `fetch: ${summarize(fetches)}` : '',
            `delivered: ${summarize(delivered)}`,
            `${((size / MiB) * (1000 / avgDelivered)).toFixed(1)}MiB/s`,
          ]
            .filter(Boolean)
            .join(' | ');
        }),
      ].join('\n')
    );
  }, 1_800_000);
});
