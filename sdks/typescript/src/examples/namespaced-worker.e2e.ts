import { Workflow, V0Worker } from '..';
import sleep from '../util/sleep';
import Hatchet from '../sdk';

xdescribe('e2e', () => {
  let hatchet: Hatchet;
  let worker: V0Worker;

  beforeEach(async () => {
    hatchet = Hatchet.init({
      namespace: 'dev',
    });
    worker = await hatchet.worker('example-worker');
  });

  afterEach(async () => {
    await worker.stop();
    await sleep(2000);
  });

  it('should pass a simple workflow', async () => {
    let invoked = 0;
    const start = new Date();

    const workflow: Workflow = {
      id: 'namespaced-e2e-workflow',
      description: 'test',
      on: {
        event: 'user:create-namespaced',
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

    await hatchet.event.push('user:create-namespaced', {
      test: 'test',
    });

    await sleep(10000);

    console.log('invoked', invoked);

    expect(invoked).toEqual(2);
  }, 60000);
});
