import { Workflow } from '..';
import sleep from '../util/sleep';
import Hatchet from '../sdk';

xdescribe('fanout-e2e', () => {
  it('should pass a fanout workflow', async () => {
    let invoked = 0;
    const start = new Date();
    const parentWorkflow: Workflow = {
      id: 'parent-workflow',
      description: 'simple example for spawning child workflows',
      steps: [
        {
          name: 'parent-spawn',
          timeout: '10s',
          run: async (ctx) => {
            const ref = ctx.spawnWorkflow('child-workflow', { input: 'child-input' });

            const res = await ref.result();
            console.log('spawned workflow result:', res);
            invoked += 1;
            return { spawned: [res] };
          },
        },
      ],
    };
    const childWorkflow: Workflow = {
      id: 'child-workflow',
      description: 'simple example for spawning child workflows',
      steps: [
        {
          name: 'child-work',
          run: async (ctx) => {
            const { input } = ctx.workflowInput();
            await sleep(1000);
            invoked += 1;
            console.log('child workflow input:', input);
            return { 'child-output': 'results' };
          },
        },
      ],
    };

    const hatchet = Hatchet.init();
    const worker = await hatchet.worker('fanout-worker');

    console.log('registering workflow...');
    await worker.registerWorkflow(parentWorkflow);
    await worker.registerWorkflow(childWorkflow);

    void worker.start();

    console.log('worker started.');

    await sleep(5000);

    console.log('running workflow...');

    await hatchet.admin.runWorkflow('parent-workflow', { input: 'parent-input' });

    await sleep(10000);

    console.log('invoked', invoked);

    expect(invoked).toEqual(2);

    await worker.stop();
  }, 120000);
});
