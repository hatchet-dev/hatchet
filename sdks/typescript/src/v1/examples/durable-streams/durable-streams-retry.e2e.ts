import { makeE2EClient, makeTestScope, startWorker, stopWorker } from '../__e2e__/harness';
import { retryingProgressTask } from './workflow';

type ProgressMessage = { attempt?: number; chunk?: number; done?: boolean; ready?: boolean };

const decode = (payload: Uint8Array): ProgressMessage =>
  JSON.parse(new TextDecoder().decode(payload));

// knows only the topic: not the run, its retry count, or where any attempt starts
async function readUntilDone(events: AsyncIterator<{ payload: Uint8Array }>) {
  const received: ProgressMessage[] = [];

  for (let next = await events.next(); !next.done; next = await events.next()) {
    const msg = decode(next.value.payload);

    if (msg.done) {
      await events.return?.();
      break;
    }

    if (!msg.ready) {
      received.push(msg);
    }
  }

  return received;
}

const chunksOf = (attempt: number) => [0, 1, 2].map((chunk) => ({ attempt, chunk }));

describe('durable-streams-e2e retries', () => {
  const hatchet = makeE2EClient();
  let worker: Awaited<ReturnType<typeof startWorker>> | undefined;

  beforeAll(async () => {
    worker = await startWorker({
      client: hatchet,
      name: 'durable-streams-retry-e2e',
      workflows: [retryingProgressTask],
    });
  });

  afterAll(async () => {
    await stopWorker(worker);
  });

  it('readers follow a retried task by topic alone, and new readers skip the failed attempt', async () => {
    const topic = makeTestScope('durable_streams_retry');

    // reading the marker back proves the subscription is live before the run's resets
    const live = hatchet.streams.events(topic)[Symbol.asyncIterator]();
    await hatchet.streams.publish(topic, JSON.stringify({ ready: true }));
    expect(decode((await live.next()).value.payload)).toEqual({ ready: true });

    const liveReading = readUntilDone(live);

    await retryingProgressTask.run({ topic });

    // a reset doesn't affect a reader already subscribed, so it sees both attempts
    expect(await liveReading).toEqual([...chunksOf(0), ...chunksOf(1)]);

    const late = hatchet.streams.events(topic)[Symbol.asyncIterator]();
    expect(await readUntilDone(late)).toEqual(chunksOf(1));
  }, 60_000);
});
