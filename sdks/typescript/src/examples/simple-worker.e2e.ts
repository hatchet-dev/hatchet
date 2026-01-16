import { Workflow, Worker } from '..';
import sleep from '../util/sleep';
import Hatchet from '../sdk';

describe('e2e', () => {
  let hatchet: Hatchet;
  let worker: Worker;

  beforeEach(async () => {
    hatchet = Hatchet.init();
    worker = await hatchet.worker('example-worker');
  });

  afterEach(async () => {
    await worker.stop();
    await sleep(2000);
  });

  it('should pass a simple workflow', async () => {
    let invoked = 0;
    const start = new Date();
    const runId = `${Date.now()}-${Math.random().toString(16).slice(2)}`;
    const workflowId = `simple-e2e-workflow-${runId}`.toLowerCase();
    const eventName = `user:create-simple-${runId}`;

    const workflow: Workflow = {
      id: workflowId,
      description: 'test',
      on: {
        event: eventName,
      },
      steps: [
        {
          name: 'step1',
          run: async (ctx) => {
            console.log('starting step1 with the following input', ctx.workflowInput());
            console.log(`took ${new Date().getTime() - start.getTime()}ms`);
            invoked += 1;
            return { step1: 'step1 results!' };
          },
        },
        {
          name: 'step2',
          parents: ['step1'],
          run: (ctx) => {
            console.log(`step 1 -> 2 took ${new Date().getTime() - start.getTime()}ms`);
            console.log('executed step2 after step1 returned ', ctx.stepOutput('step1'));
            invoked += 1;
            return { step2: 'step2 results!' };
          },
        },
      ],
    };

    console.log('registering workflow...');
    await worker.registerWorkflow(workflow);

    void worker.start();

    console.log('worker started.');

    await sleep(5000);

    console.log('pushing event...');

    await hatchet.events.push(eventName, {
      test: 'test',
    });

    // Wait until both steps have executed. We avoid fixed sleeps because
    // CI timing can vary and old queued jobs can otherwise make this flaky.
    const deadline = Date.now() + 30000;
    while (Date.now() < deadline && invoked < 2) {
      await sleep(250);
    }

    console.log('invoked', invoked);

    expect(invoked).toEqual(2);
  }, 60000);
});
