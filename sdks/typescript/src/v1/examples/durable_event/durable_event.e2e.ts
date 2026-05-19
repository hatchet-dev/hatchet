import sleep from '@hatchet/util/sleep';
import { makeE2EClient } from '../__e2e__/harness';
import { durableEvent, durableEventWithFilter, EVENT_KEY } from './workflow';

describe('durable-event-e2e', () => {
  const hatchet = makeE2EClient();

  it('waits for a user event', async () => {
    const ref = await durableEvent.runNoWait({});

    let finished = false;
    const resultPromise = ref.output.finally(() => {
      finished = true;
    });

    const eventPusher = (async () => {
      await sleep(2000);
      for (let i = 0; i < 30 && !finished; i += 1) {
        await hatchet.events.push(EVENT_KEY, { userId: '1234' });
        await sleep(200);
      }
    })();

    const result = await resultPromise;
    await eventPusher.catch(() => undefined);

    expect(result.Value).toBe('done');
  }, 120_000);

  it('waits for a user event with filter', async () => {
    const ref = await durableEventWithFilter.runNoWait({});

    let finished = false;
    const resultPromise = ref.output.finally(() => {
      finished = true;
    });

    const eventPusher = (async () => {
      await sleep(2000);
      for (let i = 0; i < 30 && !finished; i += 1) {
        await hatchet.events.push(EVENT_KEY, { userId: '1234' });
        await sleep(200);
      }
    })();

    const result = await resultPromise;
    await eventPusher.catch(() => undefined);

    expect(result.Value).toBe('done');
  }, 120_000);
});
