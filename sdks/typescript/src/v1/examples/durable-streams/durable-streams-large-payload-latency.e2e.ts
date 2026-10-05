import { randomBytes } from 'crypto';
import { performance } from 'perf_hooks';
import { StreamEvent } from '@hatchet/v1/client/features/streams';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';

const MiB = 1024 * 1024;

// the server's per-message gRPC limit (MaxStreamMessagePayloadBytes); anything larger is uploaded
const maxInlinePayloadBytes = 4 * MiB - 5 * 1024;

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

function percentile(sorted: number[], p: number): number {
  return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * p))];
}

function summarize(values: number[]): string {
  const sorted = [...values].sort((a, b) => a - b);
  const avg = values.reduce((sum, v) => sum + v, 0) / values.length;
  return `avg=${avg.toFixed(0)} p50=${percentile(sorted, 0.5).toFixed(0)} max=${sorted[sorted.length - 1].toFixed(0)}`;
}

function formatSize(bytes: number): string {
  return bytes % MiB === 0 ? `${bytes / MiB}MiB` : `${(bytes / MiB).toFixed(2)}MiB`;
}

// a benchmark, run only when asked for (test_durable_streams.sh --bench)
const describeBenchmark = process.env.HATCHET_E2E_BENCHMARKS ? describe : describe.skip;

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
      const abort = new AbortController();
      let onEvent: ((ev: StreamEvent) => void) | undefined;

      const subscriber = (async () => {
        for await (const ev of hatchet.streams.events(topic, { signal: abort.signal })) {
          onEvent?.(ev);
        }
      })();

      const nextEvent = () =>
        new Promise<StreamEvent>((resolve) => {
          onEvent = resolve;
        });

      // the subscription is live once a first message comes through
      const ready = nextEvent();
      await hatchet.streams.publish(topic, 'ready');
      await ready;

      // random bytes, so compression can't hide the cost of large messages
      const payload = new Uint8Array(randomBytes(size));
      const samples: Sample[] = [];

      // one unmeasured message absorbs topic and connection warmup at this size
      for (let i = 0; i < count + 1; i += 1) {
        trace = {};
        const delivered = nextEvent();

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

      abort.abort();
      await subscriber;

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
          const avgDelivered = delivered.reduce((sum, v) => sum + v, 0) / delivered.length;

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
