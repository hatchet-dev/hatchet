import { createGrpcClient } from '@hatchet/util/grpc-helpers';
import { V1StreamsDefinition } from '@hatchet/protoc/v1/streams';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

// deterministic varying payload size (2KB-30KB) so catch-up needs several
// StreamMessage frames per page, not just several DB pages. This is an
// integration-level check that a large, real catch-up over the wire works --
// the precise regression test for chunkStreamEntries's encoded-vs-raw-payload
// sizing bug (see internal/services/streams/batch.go) is a Go unit test
// (batch_test.go), since reproducing it needs tens of thousands of
// near-payload-free entries, impractical to publish one at a time here.
function payloadSizeFor(index: number): number {
  const cycle = index % 10;
  return 2_000 + cycle * 3_000;
}

function makePayload(index: number): string {
  const size = payloadSizeFor(index);
  // `${index}:` prefix makes every message identifiable and its expected
  // length checkable regardless of the padding character run that follows
  const prefix = `${index}:`;
  return prefix + 'x'.repeat(Math.max(0, size - prefix.length));
}

describe('durable-streams-e2e catch-up batching', () => {
  const hatchet = makeE2EClient();

  it('replays a large, mixed-size backlog in full, in order, across multiple StreamMessage frames', async () => {
    const topic = makeTestScope('durable_streams_catchup_batching');
    const messageCount = 1000;

    // populate the topic fully before any consumer attaches, forcing the
    // Subscribe call below to do a real multi-page, multi-frame catch-up
    // rather than a live tail
    for (let i = 0; i < messageCount; i += 1) {
      await hatchet.streams.publish(topic, makePayload(i));
    }

    // talk to the raw Subscribe RPC directly (bypassing events()) so we can
    // count how many StreamMessage frames the catch-up actually took --
    // proving chunking was exercised, not just that everything happened to
    // fit in one frame
    const { client } = createGrpcClient(hatchet.config, V1StreamsDefinition);

    const stream = client.subscribe({ namespace: '', topic, cursor: undefined });

    let frameCount = 0;
    let maxEntriesPerFrame = 0;
    const received: string[] = [];

    for await (const frame of stream) {
      if (frame.hangup) break;
      if (frame.entries.length === 0) continue;

      frameCount += 1;
      maxEntriesPerFrame = Math.max(maxEntriesPerFrame, frame.entries.length);

      for (const entry of frame.entries) {
        received.push(decode(entry.payload));
      }

      if (received.length >= messageCount) break;
    }

    expect(received).toHaveLength(messageCount);

    // every message arrived, in publish order, with its exact payload intact
    received.forEach((payload, i) => {
      expect(payload).toEqual(makePayload(i));
    });

    // the whole backlog fitting in one frame would mean this test isn't
    // actually exercising chunkStreamEntries's splitting logic
    expect(frameCount).toBeGreaterThan(1);
    // and no single frame should come anywhere near the ~4mb gRPC limit that
    // maxStreamMessageBatchBytes exists to stay under
    expect(maxEntriesPerFrame).toBeLessThan(messageCount);
  }, 120_000);
});
