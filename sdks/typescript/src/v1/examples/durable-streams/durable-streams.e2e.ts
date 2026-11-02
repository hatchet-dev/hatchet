import { Status } from 'nice-grpc';
import { makeE2EClient, makeTestScope } from '../__e2e__/harness';

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

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

  it('a subscribe starts from beginning of queue, even when publish happened prior', async () => {
    const topic = makeTestScope('durable_streams_from_beginning');

    // publish returns once the message is committed
    await hatchet.streams.publish(topic, 'published-before-consume');

    const events = hatchet.streams.subscribe(topic);

    const publishedAfter = ['live-1', 'live-2'];
    setTimeout(() => {
      publishedAfter.forEach((msg) => {
        void hatchet.streams.publish(topic, msg);
      });
    }, 200);

    const received = await collect(events, publishedAfter.length + 1);

    expect(received.map((e) => decode(e.payload))).toEqual([
      'published-before-consume',
      'live-1',
      'live-2',
    ]);
  }, 30_000);

  it('resumes from a cursor read off a previously received event, with no re-delivery and no gap', async () => {
    const topic = makeTestScope('durable_streams_resume_cursor');
    const messages = ['one', 'two', 'three', 'four'];

    // A cursor is purely a client-side construction -- publish() doesn't
    // return one; the only way to get one is to read it off a message
    // actually received from subscribe(). Open the consumer before publishing
    // (no cursor = start from now) so every message below is observed live.
    const events = hatchet.streams.subscribe(topic);

    messages.forEach((msg) => {
      void hatchet.streams.publish(topic, msg);
    });

    const firstTwo = await collect(events, 2);
    expect(firstTwo.map((e) => decode(e.payload))).toEqual(['one', 'two']);

    // resume from the 2nd received event's own cursor -- expect "three", "four"
    const checkpoint = firstTwo[firstTwo.length - 1].cursor;

    const rest = await collect(hatchet.streams.subscribe(topic, { cursor: checkpoint }), 2);
    expect(rest.map((e) => decode(e.payload))).toEqual(['three', 'four']);
  }, 30_000);

  it('reset starts new subscribers after the latest message, while existing ones carry on', async () => {
    const topic = makeTestScope('durable_streams_reset');

    await hatchet.streams.publish(topic, 'before-reset');

    const existing = hatchet.streams.subscribe(topic)[Symbol.asyncIterator]();
    expect(decode((await existing.next()).value.payload)).toEqual('before-reset');

    await hatchet.streams.reset(topic);
    await hatchet.streams.publish(topic, 'after-reset');

    const [fresh] = await collect(hatchet.streams.subscribe(topic), 1);
    expect(decode(fresh.payload)).toEqual('after-reset');

    expect(decode((await existing.next()).value.payload)).toEqual('after-reset');
    await existing.return?.();
  }, 30_000);

  it('reset to a cursor starts new subscribers after that message', async () => {
    const topic = makeTestScope('durable_streams_reset_cursor');

    await hatchet.streams.publish(topic, 'one');
    await hatchet.streams.publish(topic, 'two');
    await hatchet.streams.publish(topic, 'three');

    const [first] = await collect(hatchet.streams.subscribe(topic), 1);
    await hatchet.streams.reset(topic, { cursor: first.cursor });

    const fresh = await collect(hatchet.streams.subscribe(topic), 2);
    expect(fresh.map((e) => decode(e.payload))).toEqual(['two', 'three']);
  }, 30_000);

  it('rejects a reset to a cursor from another topic', async () => {
    const topic = makeTestScope('durable_streams_reset_foreign_cursor');
    const other = makeTestScope('durable_streams_reset_foreign_cursor_other');

    await hatchet.streams.publish(other, 'elsewhere');
    const [foreign] = await collect(hatchet.streams.subscribe(other), 1);

    await expect(hatchet.streams.reset(topic, { cursor: foreign.cursor })).rejects.toMatchObject({
      code: Status.INVALID_ARGUMENT,
    });
  }, 30_000);

  it('isolates messages by namespace', async () => {
    const topic = makeTestScope('durable_streams_namespace');
    const namespaceA = 'ns-a';
    const namespaceB = 'ns-b';

    // committed on return, so a consumer scoped to namespaceB could see it if scoping were broken
    await hatchet.streams.publish(topic, 'from-a', { namespace: namespaceA });

    const events = hatchet.streams.subscribe(topic, { namespace: namespaceB });

    setTimeout(() => {
      void hatchet.streams.publish(topic, 'from-b', { namespace: namespaceB });
    }, 200);

    const [received] = await collect(events, 1);

    expect(decode(received.payload)).toEqual('from-b');
  }, 30_000);

  it('topic metadata reports the newest message, which resumes a subscription after it', async () => {
    const topic = makeTestScope('durable_streams_metadata');

    for (const msg of ['first', 'second', 'third']) {
      await hatchet.streams.publish(topic, msg);
    }

    const md = await hatchet.streams.topicMetadata(topic);

    expect(md).toMatchObject({ namespace: '', topic, messageCount: 3 });
    expect(md.tenantId).toEqual(hatchet.config.tenant_id);
    expect(md.lastPublishedAt).toBeInstanceOf(Date);

    const events = await collect(hatchet.streams.subscribe(topic), 3);
    expect(md.latestCursor).toEqual(events[2].cursor);

    await expect(
      hatchet.streams.topicMetadata(makeTestScope('durable_streams_missing'))
    ).rejects.toMatchObject({
      code: Status.NOT_FOUND,
    });
  });
});
