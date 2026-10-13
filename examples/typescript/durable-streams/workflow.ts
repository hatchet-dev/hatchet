import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import { hatchet } from '../hatchet-client';

// > Publishing
export type ProgressInput = {
  topic: string;
};

export const progressTask = hatchet.task({
  name: 'durable-streams-progress',
  fn: async (input: ProgressInput) => {
    const total = 5;

    for (let step = 1; step <= total; step += 1) {
      await sleep(500);
      await hatchet.streams.publish(input.topic, JSON.stringify({ step, total }));
    }

    return { steps: total };
  },
});

// > Reset on retry
export const retryingProgressTask = hatchet.task({
  name: 'durable-streams-retrying-progress',
  retries: 1,
  fn: async (input: ProgressInput, ctx) => {
    // readers that subscribe from here on skip a failed attempt's messages
    await hatchet.streams.reset(input.topic);

    for (let chunk = 0; chunk < 3; chunk += 1) {
      await hatchet.streams.publish(
        input.topic,
        JSON.stringify({ attempt: ctx.retryCount(), chunk })
      );
    }

    if (ctx.retryCount() === 0) {
      throw new Error('intentional failure before the final message');
    }

    await hatchet.streams.publish(input.topic, JSON.stringify({ done: true }));

    return { chunks: 3 };
  },
});
