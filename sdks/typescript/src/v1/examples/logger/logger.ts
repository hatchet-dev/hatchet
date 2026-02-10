import { hatchet } from '../hatchet-client';

const sleep = (ms: number) =>
  new Promise((resolve) => {
    setTimeout(resolve, ms);
  });

// > Logger

const workflow = hatchet.workflow({
  name: 'logger-example',
  description: 'test',
  on: {
    event: 'user:create',
  },
});

workflow.task({
  name: 'logger-step1',
  fn: async (_, ctx) => {
    // log in a for loop
    // eslint-disable-next-line no-plusplus
    for (let i = 0; i < 10; i++) {
      ctx.logger.info(`log message ${i}`);
      await sleep(200);
    }

    return { step1: 'completed step run' };
  },
});

// !!

async function main() {
  const worker = await hatchet.worker('logger-worker', {
    slots: 1,
    workflows: [workflow],
  });
  await worker.start();
}

main();
