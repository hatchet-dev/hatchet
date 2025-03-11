import Hatchet from '../sdk';
import { Context } from '../step';

const hatchet: Hatchet = Hatchet.init();

async function main() {
  const worker = await hatchet.worker('test-playground');

  await worker.registerWorkflow({
    id: 'playground-ts',
    description: 'desc',
    on: {
      event: 'test1',
    },
    steps: [
      {
        name: 'test1-step1',
        run: (ctx: Context<any, any>) => {
          const playground = ctx.playground('test1', 'default');

          return { step1: playground, name: ctx.stepName(), workflowRunId: ctx.workflowRunId() };
        },
      },
    ],
  });

  await worker.start();
}

main();
