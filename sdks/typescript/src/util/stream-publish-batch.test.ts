import HatchetError from '@util/errors/hatchet-error';
import {
  MAX_PUBLISH_BATCH_BYTES,
  MAX_PUBLISH_BATCH_ENTRIES,
  StreamPublishQueue,
  validateBatchTimeout,
} from './stream-publish-batch';

type Item = { id: number; bytes: number };

const item = (id: number, bytes = 1): Item => ({ id, bytes });
const ids = (batch: Item[] | undefined) => batch?.map((b) => b.id);
const newQueue = () => new StreamPublishQueue<Item>((i) => i.bytes);

describe('StreamPublishQueue', () => {
  beforeEach(() => {
    jest.useFakeTimers();
  });

  afterEach(() => {
    jest.useRealTimers();
  });

  // records next()'s batch once it resolves, so a test can check it is still pending
  function nextBatch(queue: StreamPublishQueue<Item>) {
    const result: { batch?: Item[] } = {};
    void queue.next().then((batch) => {
      result.batch = batch;
    });
    return result;
  }

  it('sends a publish without a batch timeout at once', async () => {
    const queue = newQueue();
    queue.push(item(1));

    expect(ids(await queue.next())).toEqual([1]);
  });

  it('holds a publish for its batch timeout, sending what joined it meanwhile', async () => {
    const queue = newQueue();
    queue.push(item(1), 100);
    const result = nextBatch(queue);

    await jest.advanceTimersByTimeAsync(50);
    queue.push(item(2), 100);
    await jest.advanceTimersByTimeAsync(49);
    expect(result.batch).toBeUndefined();

    await jest.advanceTimersByTimeAsync(1);
    expect(ids(result.batch)).toEqual([1, 2]);
  });

  it('sends early when a later publish has a shorter timeout', async () => {
    const queue = newQueue();
    queue.push(item(1), 1000);
    const result = nextBatch(queue);

    await jest.advanceTimersByTimeAsync(10);
    queue.push(item(2));
    await jest.advanceTimersByTimeAsync(0);
    expect(ids(result.batch)).toEqual([1, 2]);
  });

  it('waits out a timeout longer than one timer can hold', async () => {
    const day = 24 * 60 * 60 * 1000;
    const queue = newQueue();
    queue.push(item(1), 30 * day);
    const result = nextBatch(queue);

    await jest.advanceTimersByTimeAsync(29 * day);
    expect(result.batch).toBeUndefined();
    expect(jest.getTimerCount()).toBe(1);

    await jest.advanceTimersByTimeAsync(day);
    expect(ids(result.batch)).toEqual([1]);
  });

  it('sends a full request without waiting out the timeout', async () => {
    const queue = newQueue();
    for (let i = 0; i < MAX_PUBLISH_BATCH_ENTRIES + 1; i += 1) {
      queue.push(item(i), 60_000);
    }

    expect(await queue.next()).toHaveLength(MAX_PUBLISH_BATCH_ENTRIES);
    expect(queue.size).toBe(1);
  });

  it('stops a request at the byte limit, but always sends at least one publish', async () => {
    const queue = newQueue();
    queue.push(item(1, MAX_PUBLISH_BATCH_BYTES));
    queue.push(item(2));

    expect(ids(await queue.next())).toEqual([1]);
    expect(ids(await queue.next())).toEqual([2]);
  });
});

describe('validateBatchTimeout', () => {
  it.each([undefined, 0, 50, 30 * 24 * 60 * 60 * 1000])('accepts %p', (ms) => {
    expect(() => validateBatchTimeout(ms)).not.toThrow();
  });

  it.each([-1, NaN, Infinity])('rejects %p', (ms) => {
    expect(() => validateBatchTimeout(ms)).toThrow(HatchetError);
  });
});
