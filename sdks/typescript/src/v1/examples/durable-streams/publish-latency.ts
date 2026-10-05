import { performance } from 'perf_hooks';
import { HatchetClient } from '@hatchet/v1';

export type PublishLatencyStats = {
  count: number;
  avgMs: number;
  p50Ms: number;
  p99Ms: number;
  maxMs: number;
  throughputPerSec: number;
};

function percentile(sorted: number[], p: number): number {
  return sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * p))];
}

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
  const sorted = [...durationsMs].sort((a, b) => a - b);

  return {
    count,
    avgMs: durationsMs.reduce((sum, d) => sum + d, 0) / count,
    p50Ms: percentile(sorted, 0.5),
    p99Ms: percentile(sorted, 0.99),
    maxMs: sorted[sorted.length - 1],
    throughputPerSec: (count / totalMs) * 1000,
  };
}

export function formatPublishLatency(stats: PublishLatencyStats): string {
  const fmt = (ms: number) => ms.toFixed(2);

  return (
    `avg=${fmt(stats.avgMs)}ms p50=${fmt(stats.p50Ms)}ms p99=${fmt(stats.p99Ms)}ms ` +
    `max=${fmt(stats.maxMs)}ms throughput=${fmt(stats.throughputPerSec)}/s`
  );
}
