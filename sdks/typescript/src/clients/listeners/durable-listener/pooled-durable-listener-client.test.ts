import { DurableEventStreamable } from './pooled-durable-listener-client';

const dummyListener: AsyncIterable<any> = (async function* gen() {
  // never yields
})();

describe('DurableEventStreamable.get cancellation', () => {
  it('rejects with AbortError and runs cleanup when aborted', async () => {
    const cleanup = jest.fn();
    const s = new DurableEventStreamable(dummyListener, 'task', 'key', 'sub-1', cleanup);
    const ac = new AbortController();

    const p = s.get({ signal: ac.signal });
    ac.abort();

    await expect(p).rejects.toMatchObject({ name: 'AbortError' });
    expect(cleanup).toHaveBeenCalledTimes(1);
  });

  it('resolves on response and runs cleanup once', async () => {
    const cleanup = jest.fn();
    const s = new DurableEventStreamable(dummyListener, 'task', 'key', 'sub-1', cleanup);

    const event: any = { taskId: 'task', signalKey: 'key', data: '{}' };
    setTimeout(() => s.responseEmitter.emit('response', event), 0);

    await expect(s.get()).resolves.toEqual(event);
    expect(cleanup).toHaveBeenCalledTimes(1);
  });
});
