import { makeE2EClient, makeTestScope } from '../__e2e__/harness';

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

describe('durable-streams-e2e latency', () => {
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

    latenciesMs.sort((a, b) => a - b);
    const p50 = latenciesMs[Math.floor(latenciesMs.length * 0.5)];
    const p99 = latenciesMs[Math.floor(latenciesMs.length * 0.99)];
    const max = latenciesMs[latenciesMs.length - 1];

    console.log(
      `durable streams e2e latency over ${messageCount} messages: p50=${p50}ms p99=${p99}ms max=${max}ms`
    );

    expect(latenciesMs).toHaveLength(messageCount);

    // a generous bound -- this guards against a pathological regression (e.g.
    // every message tripping the controller's producer-gap wait) rather than
    // asserting a strict performance SLA that would be flaky in CI
    expect(p99).toBeLessThan(10_000);
  }, 180_000);
});
