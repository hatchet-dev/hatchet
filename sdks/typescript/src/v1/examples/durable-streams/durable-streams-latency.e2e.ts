import { makeE2EClient, makeTestScope } from '../__e2e__/harness';
import { describeBenchmark, formatLatency, latencyStats } from './benchmark';

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

describeBenchmark('durable-streams-e2e latency', () => {
  const hatchet = makeE2EClient();

  it('measures end-to-end publish-to-receive latency across a large number of messages', async () => {
    const topic = makeTestScope('durable_streams_latency');
    const messageCount = 400;

    const events = hatchet.streams.events(topic);

    // publishes run in the background while we consume below, so this
    // measures real publish -> receive latency per message rather than
    // latency after the whole batch has already landed
    const publishAll = (async () => {
      for (let i = 0; i < messageCount; i += 1) {
        await hatchet.streams.publish(topic, String(Date.now()));
      }
    })();

    const latenciesMs: number[] = [];
    for await (const event of events) {
      const sentAt = Number(decode(event.payload));
      latenciesMs.push(Date.now() - sentAt);
      if (latenciesMs.length >= messageCount) break;
    }

    await publishAll;

    const stats = latencyStats(latenciesMs);

    console.log(
      `durable streams e2e latency over ${messageCount} messages: ${formatLatency(stats, 0)}`
    );

    expect(stats.count).toBe(messageCount);

    // a generous bound -- this guards against a pathological regression (e.g.
    // every message waiting for the tail poller's fallback tick) rather than
    // asserting a strict performance SLA that would be flaky in CI
    expect(stats.p99Ms).toBeLessThan(10_000);
  }, 180_000);
});
