import sleep from '../../../util/sleep';
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
// !!
