import {
  MAX_PUBLISH_BATCH_BYTES,
  MAX_PUBLISH_BATCH_ENTRIES,
  StreamPublishQueue,
} from './stream-publish-batch';

describe('StreamPublishQueue', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  const item = (id: number, timeoutMs: number, bytes = 1) => ({
    id,
    bytes,
    sendBy: Date.now() + timeoutMs,
  });

  it('sends a publish without a batch timeout at once', async () => {
    const queue = new StreamPublishQueue<ReturnType<typeof item>>();
    queue.push(item(1, 0));

    await expect(queue.next()).resolves.toEqual([expect.objectContaining({ id: 1 })]);
  });

  it('holds a publish for its batch timeout, sending what joined it meanwhile', async () => {
    const queue = new StreamPublishQueue<ReturnType<typeof item>>();
    queue.push(item(1, 100));

    let batch: Array<{ id: number }> | undefined;
    void queue.next().then((b) => {
      batch = b;
    });

    await jest.advanceTimersByTimeAsync(50);
    queue.push(item(2, 100));
    await jest.advanceTimersByTimeAsync(49);
    expect(batch).toBeUndefined();

    await jest.advanceTimersByTimeAsync(1);
    expect(batch?.map((b) => b.id)).toEqual([1, 2]);
  });

  it('sends early when a later publish has a shorter timeout', async () => {
    const queue = new StreamPublishQueue<ReturnType<typeof item>>();
    queue.push(item(1, 1000));

    let batch: Array<{ id: number }> | undefined;
    void queue.next().then((b) => {
      batch = b;
    });

    await jest.advanceTimersByTimeAsync(10);
    queue.push(item(2, 0));
    await jest.advanceTimersByTimeAsync(0);
    expect(batch?.map((b) => b.id)).toEqual([1, 2]);
  });

  it('sends a full request without waiting out the timeout', async () => {
    const queue = new StreamPublishQueue<ReturnType<typeof item>>();
    for (let i = 0; i < MAX_PUBLISH_BATCH_ENTRIES + 1; i += 1) {
      queue.push(item(i, 60_000));
    }

    const batch = await queue.next();
    expect(batch).toHaveLength(MAX_PUBLISH_BATCH_ENTRIES);
    expect(queue.size).toBe(1);
  });

  it('stops a request at the byte limit, but always sends at least one publish', async () => {
    const queue = new StreamPublishQueue<ReturnType<typeof item>>();
    queue.push(item(1, 0, MAX_PUBLISH_BATCH_BYTES));
    queue.push(item(2, 0, 1));

    await expect(queue.next()).resolves.toEqual([expect.objectContaining({ id: 1 })]);
    await expect(queue.next()).resolves.toEqual([expect.objectContaining({ id: 2 })]);
  });
});
