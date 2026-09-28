import sleep from '@hatchet/util/sleep';
import { makeE2EClient, makeTestScope, poll } from '../__e2e__/harness';

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

async function collect<T>(iter: AsyncIterable<T>, count: number): Promise<T[]> {
  const out: T[] = [];
  for await (const item of iter) {
    out.push(item);
    if (out.length >= count) break;
  }
  return out;
}

describe('durable-streams-e2e cancellation', () => {
  const hatchet = makeE2EClient();

  it('breaking a for-await loop stops consumption without losing messages -- they are still there for a later reader', async () => {
    const topic = makeTestScope('durable_streams_cancel_break');

    const events = hatchet.streams.events(topic);

    void hatchet.streams.publish(topic, 'one');
    void hatchet.streams.publish(topic, 'two');
    void hatchet.streams.publish(topic, 'three');

    let lastCursor = '';
    const received: string[] = [];

    for await (const event of events) {
      received.push(decode(event.payload));
      lastCursor = event.cursor;
      if (received.length === 1) {
        // "hang up" -- the idiomatic way to stop consuming early
        break;
      }
    }

    expect(received).toEqual(['one']);

    // 'two' and 'three' are unaffected by us hanging up on the first
    // consumer -- durable topics don't get drained by a reader, so a fresh
    // subscribe from the last cursor we actually saw picks up right after it
    const rest = await collect(hatchet.streams.events(topic, { cursor: lastCursor }), 2);
    expect(rest.map((e) => decode(e.payload))).toEqual(['two', 'three']);
  }, 30_000);

  it('aborting an idle subscription returns immediately, instead of waiting for the next message', async () => {
    const topic = makeTestScope('durable_streams_cancel_abort_idle');
    const controller = new AbortController();

    // nothing is ever published to this topic -- without a working cancel
    // signal this would only resolve once the server's own idle hangup timer
    // fires (30 minutes, see subscribeIdleHangupTimeout), which is exactly
    // the problem a cancel signal exists to avoid
    const iterator = hatchet.streams
      .events(topic, { signal: controller.signal })
      [Symbol.asyncIterator]();
    const nextPromise = iterator.next();

    // give the Subscribe call time to actually reach the server before we
    // cancel it, so this exercises a real in-flight abort, not a no-op on a
    // call that hadn't started yet
    await sleep(500);

    const abortedAt = Date.now();
    controller.abort();

    const result = await nextPromise;
    const elapsedMs = Date.now() - abortedAt;

    expect(result.done).toBe(true);
    expect(elapsedMs).toBeLessThan(5_000);
  }, 15_000);

  it('aborting mid-stream stops delivery of anything published after the abort', async () => {
    const topic = makeTestScope('durable_streams_cancel_abort_mid_stream');
    const controller = new AbortController();

    await hatchet.streams.publish(topic, 'before-abort');

    const received: string[] = [];
    const consumePromise = (async () => {
      for await (const event of hatchet.streams.events(topic, { signal: controller.signal })) {
        received.push(decode(event.payload));
      }
    })();

    // wait until the one message published above has actually been received,
    // proving the subscription is live, before cancelling it
    await poll(() => Promise.resolve(received.length), {
      shouldStop: (count) => count >= 1,
      timeoutMs: 10_000,
      label: 'before-abort message to be received',
    });
    controller.abort();
    await consumePromise;

    await hatchet.streams.publish(topic, 'after-abort');
    await sleep(500);

    expect(received).toEqual(['before-abort']);
  }, 30_000);
});
