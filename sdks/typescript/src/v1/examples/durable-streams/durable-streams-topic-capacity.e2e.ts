import { monitorEventLoopDelay, performance } from 'perf_hooks';
import { HatchetClient } from '@hatchet/v1';
import { makeTestScope } from '../__e2e__/harness';
import { describeBenchmark, formatLatency, latencyStats } from './benchmark';

const topicCount = Number(process.env.DURABLE_STREAMS_TOPICS ?? 100);
const messagesPerTopic = Math.max(1, Number(process.env.DURABLE_STREAMS_MESSAGES_PER_TOPIC ?? 10));
const connectionCount = Number(process.env.DURABLE_STREAMS_CONNECTIONS ?? 1);
// one tenant publishing to every topic at once trips the engine's per-tenant database limit
const maxInflightPublishes = Number(process.env.DURABLE_STREAMS_MAX_INFLIGHT_PUBLISHES ?? 200);
const timeoutMs = Number(process.env.DURABLE_STREAMS_CAPACITY_TIMEOUT_MS ?? 300_000);

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

// grpc-js shares one connection per target across a module instance, so each client needs its own copy
function makeIsolatedClient(): HatchetClient {
  let client: HatchetClient | undefined;
  jest.isolateModules(() => {
    // eslint-disable-next-line @typescript-eslint/no-require-imports
    client = require('../__e2e__/harness').makeE2EClient();
  });
  return client!;
}

function concurrencyLimit(max: number) {
  let active = 0;
  const waiting: (() => void)[] = [];

  return async <T>(run: () => Promise<T>): Promise<T> => {
    while (active >= max) {
      await new Promise<void>((resolve) => {
        waiting.push(resolve);
      });
    }

    active += 1;

    try {
      return await run();
    } finally {
      active -= 1;
      waiting.shift()?.();
    }
  };
}

describeBenchmark('durable-streams-e2e topic capacity', () => {
  const clients = Array.from({ length: connectionCount }, makeIsolatedClient);

  it(
    `delivers every message, in order, to ${topicCount} concurrently subscribed topics over ${connectionCount} connection(s)`,
    async () => {
      const scope = makeTestScope('durable_streams_capacity');
      const limit = concurrencyLimit(maxInflightPublishes);
      const publishAckMs: number[] = [];
      const deliveryMs: number[] = [];

      const topics = Array.from({ length: topicCount }, (_, t) => {
        let markReady!: () => void;
        const ready = new Promise<void>((resolve) => {
          markReady = resolve;
        });

        return { topic: `${scope}_${t}`, hatchet: clients[t % connectionCount], ready, markReady };
      });

      // delivery is timed in this process, so a saturated event loop shows up as delivery latency
      const eventLoopDelay = monitorEventLoopDelay({ resolution: 10 });
      eventLoopDelay.enable();

      const setupStart = performance.now();

      const received = topics.map(async ({ topic, hatchet, markReady }) => {
        const indexes: number[] = [];

        for await (const event of hatchet.streams.subscribe(topic)) {
          const text = decode(event.payload);

          if (text === 'ready') {
            markReady();
            continue;
          }

          const [index, sentAt] = text.split(':').map(Number);
          deliveryMs.push(performance.now() - sentAt);
          indexes.push(index);

          if (indexes.length >= messagesPerTopic) break;
        }

        return indexes;
      });

      await Promise.all(
        topics.map(({ topic, hatchet }) => limit(() => hatchet.streams.publish(topic, 'ready')))
      );
      await Promise.all(topics.map(({ ready }) => ready));
      const setupMs = performance.now() - setupStart;

      const publishStart = performance.now();

      await Promise.all(
        topics.map(async ({ topic, hatchet }) => {
          for (let i = 0; i < messagesPerTopic; i += 1) {
            await limit(async () => {
              const sentAt = performance.now();
              await hatchet.streams.publish(topic, `${i}:${sentAt}`);
              publishAckMs.push(performance.now() - sentAt);
            });
          }
        })
      );
      const publishedMs = performance.now() - publishStart;

      const results = await Promise.all(received);
      const deliveredMs = performance.now() - publishStart;
      eventLoopDelay.disable();
      const total = topicCount * messagesPerTopic;

      console.log(
        [
          `topics=${topicCount} messagesPerTopic=${messagesPerTopic} connections=${connectionCount} maxInflightPublishes=${maxInflightPublishes}`,
          `  ${topicCount} subscriptions live in ${setupMs.toFixed(0)}ms`,
          `  ${total} messages published in ${publishedMs.toFixed(0)}ms, all delivered in ${deliveredMs.toFixed(0)}ms (${((total * 1000) / deliveredMs).toFixed(1)} msg/s)`,
          `  publish ack: ${formatLatency(latencyStats(publishAckMs), 0)}`,
          `  delivery:    ${formatLatency(latencyStats(deliveryMs), 0)}`,
          `  client event loop delay: p50=${(eventLoopDelay.percentile(50) / 1e6).toFixed(0)}ms p99=${(eventLoopDelay.percentile(99) / 1e6).toFixed(0)}ms max=${(eventLoopDelay.max / 1e6).toFixed(0)}ms`,
        ].join('\n')
      );

      const expected = Array.from({ length: messagesPerTopic }, (_, i) => i);
      results.forEach((indexes) => expect(indexes).toEqual(expected));
    },
    timeoutMs
  );
});
