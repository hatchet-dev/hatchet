import { performance } from 'perf_hooks';
import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';
import { StreamEvent } from '@hatchet-dev/typescript-sdk/v1/client/features/streams';

export const MiB = 1024 * 1024;

// the server's per-message gRPC limit (MaxStreamMessagePayloadBytes); larger payloads are uploaded
export const maxInlinePayloadBytes = 4 * MiB - 5 * 1024;

// benchmarks run only when asked for (test_durable_streams.sh --bench)
export const describeBenchmark = process.env.HATCHET_E2E_BENCHMARKS ? describe : describe.skip;

export type LatencyStats = {
  count: number;
  avgMs: number;
  p50Ms: number;
  p99Ms: number;
  maxMs: number;
};

function percentile(sorted: number[], p: number): number {
  return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * p))];
}

export function latencyStats(durationsMs: number[]): LatencyStats {
  const sorted = [...durationsMs].sort((a, b) => a - b);

  return {
    count: durationsMs.length,
    avgMs: durationsMs.reduce((sum, d) => sum + d, 0) / durationsMs.length,
    p50Ms: percentile(sorted, 0.5),
    p99Ms: percentile(sorted, 0.99),
    maxMs: sorted[sorted.length - 1],
  };
}

export function formatLatency(stats: LatencyStats, digits = 2): string {
  const fmt = (ms: number) => ms.toFixed(digits);

  return `avg=${fmt(stats.avgMs)}ms p50=${fmt(stats.p50Ms)}ms p99=${fmt(stats.p99Ms)}ms max=${fmt(stats.maxMs)}ms`;
}

export function formatSize(bytes: number): string {
  if (bytes % MiB === 0) return `${bytes / MiB}MiB`;
  if (bytes % 1024 === 0) return `${bytes / 1024}KiB`;
  return `${bytes}B`;
}

export type PublishLatencyStats = LatencyStats & { throughputPerSec: number };

/**
 * Times `count` sequential publishes, after `warmupCount` unmeasured ones
 * absorb connection setup and topic creation.
 */
export async function measurePublishLatency(
  hatchet: HatchetClient,
  topic: string,
  payload: Uint8Array | string,
  count: number,
  warmupCount = 5
): Promise<PublishLatencyStats> {
  for (let i = 0; i < warmupCount; i += 1) {
    await hatchet.streams.publish(topic, payload);
  }

  const durationsMs: number[] = [];
  const start = performance.now();

  for (let i = 0; i < count; i += 1) {
    const before = performance.now();
    await hatchet.streams.publish(topic, payload);
    durationsMs.push(performance.now() - before);
  }

  const totalMs = performance.now() - start;

  return { ...latencyStats(durationsMs), throughputPerSec: (count / totalMs) * 1000 };
}

export function formatPublishLatency(stats: PublishLatencyStats): string {
  return `${formatLatency(stats)} throughput=${stats.throughputPerSec.toFixed(2)}/s`;
}

export type LiveSubscription = {
  /** resolves with the next message delivered after the call */
  nextEvent: () => Promise<StreamEvent>;
  close: () => Promise<void>;
};

/**
 * Subscribes to topic and resolves once the subscription is live, so a
 * benchmark's timings don't include subscribe setup.
 */
export async function openLiveSubscription(
  hatchet: HatchetClient,
  topic: string
): Promise<LiveSubscription> {
  const abort = new AbortController();
  let onEvent: ((ev: StreamEvent) => void) | undefined;

  const consumer = (async () => {
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

  return {
    nextEvent,
    close: async () => {
      abort.abort();
      await consumer;
    },
  };
}
