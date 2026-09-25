import sleep from '@hatchet/util/sleep';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);
const encode = (text: string) => new TextEncoder().encode(text);

// Collects up to `count` events from an async iterable, then stops consuming
// (which cancels the underlying Subscribe stream). Guards against a hung test
// with the caller's jest timeout rather than its own.
async function collect<T>(iter: AsyncIterable<T>, count: number): Promise<T[]> {
  const out: T[] = [];
  for await (const item of iter) {
    out.push(item);
    if (out.length >= count) break;
  }
  return out;
}

describe('durable-streams-e2e', () => {
  const hatchet = makeE2EClient();

  it('a message published before consuming is not replayed by default (no cursor = start from now)', async () => {
    const topic = makeTestScope('durable_streams_tail_from_now');

    await hatchet.streams.publish(topic, 'published-before-consume');

    // give the streams controller a moment to durably persist the message
    // above, so this test actually exercises "already-persisted messages are
    // skipped", not just "nothing has arrived yet"
    await sleep(500);

    const events = hatchet.streams.events(topic);

    const publishedAfter = ['live-1', 'live-2'];
    setTimeout(() => {
      publishedAfter.forEach((msg) => {
        void hatchet.streams.publish(topic, msg);
      });
    }, 200);

    const received = await collect(events, publishedAfter.length);

    expect(received.map((e) => decode(e.payload))).toEqual(publishedAfter);
  }, 30_000);

  it('resumes from a cursor read off a previously received event, with no re-delivery and no gap', async () => {
    const topic = makeTestScope('durable_streams_resume_cursor');
    const messages = ['one', 'two', 'three', 'four'];

    // A cursor is purely a client-side construction -- publish() doesn't
    // return one; the only way to get one is to read it off a message
    // actually received from events(). Open the consumer before publishing
    // (no cursor = start from now) so every message below is observed live.
    const events = hatchet.streams.events(topic);

    messages.forEach((msg) => {
      void hatchet.streams.publish(topic, msg);
    });

    const firstTwo = await collect(events, 2);
    expect(firstTwo.map((e) => decode(e.payload))).toEqual(['one', 'two']);

    // resume from the 2nd received event's own cursor -- expect "three", "four"
    const checkpoint = firstTwo[firstTwo.length - 1].cursor;

    const rest = await collect(hatchet.streams.events(topic, { cursor: checkpoint }), 2);
    expect(rest.map((e) => decode(e.payload))).toEqual(['three', 'four']);
  }, 30_000);

  it('delivers messages published after a consumer is already open (catch-up -> tail)', async () => {
    const topic = makeTestScope('durable_streams_live_delivery');

    const events = hatchet.streams.events(topic);

    setTimeout(() => {
      void hatchet.streams.publish(topic, encode('after-open'));
    }, 300);

    const [first] = await collect(events, 1);

    expect(decode(first.payload)).toEqual('after-open');
  }, 30_000);

  it('isolates messages by namespace', async () => {
    const topic = makeTestScope('durable_streams_namespace');
    const namespaceA = 'ns-a';
    const namespaceB = 'ns-b';

    await hatchet.streams.publish(topic, 'from-a', { namespace: namespaceA });

    // give namespaceA's message a moment to land, then confirm a consumer
    // scoped to namespaceB never sees it
    await sleep(500);

    const events = hatchet.streams.events(topic, { namespace: namespaceB });

    setTimeout(() => {
      void hatchet.streams.publish(topic, 'from-b', { namespace: namespaceB });
    }, 200);

    const [received] = await collect(events, 1);

    expect(decode(received.payload)).toEqual('from-b');
  }, 30_000);
});
