import { createServer } from 'node:http';
import { AxiosError } from 'axios';
import { InternalHatchetClient as Hatchet } from '@hatchet/clients/hatchet-client';
import { Workflow, Worker } from '..';
import sleep from '../util/sleep';

const port = 8369;

describe('webhooks', () => {
  let hatchet: Hatchet; // TODO upgrade to v1
  let worker: Worker;

  beforeEach(async () => {
    hatchet = Hatchet.init();
    worker = await hatchet.worker('webhook-workflow');
  });

  afterEach(async () => {
    await worker.stop();
    await sleep(2000);
  });

  it('should pass a webhook workflow', async () => {
    let invoked = 0;

    const workflow: Workflow = {
      id: 'webhook-workflow',
      description: 'test',
      on: {
        event: 'user:create-webhook',
      },
      steps: [
        {
          name: 'step1',
          run: async (ctx) => {
            console.log('step1', ctx.workflowInput());
            invoked += 1;
            return { message: `${ctx.workflowName()} results!` };
          },
        },
        {
          name: 'step2',
          parents: ['step1'],
          run: (ctx) => {
            console.log('step2', ctx.workflowInput());
            invoked += 1;
            return { message: `${ctx.workflowName()} results!` };
          },
        },
      ],
    };

    // registering workflows is not needed because it will be done automatically

    const secret = 'secret';

    console.log('registering webhook...');
    try {
      await worker.registerWebhook({
        name: 'webhook-example',
        secret,
        url: `http://localhost:${port}/webhook`,
      });
    } catch (e) {
      const axiosError = e as AxiosError;
      console.error(axiosError.response?.data, axiosError.request, axiosError.request.method);
      throw e;
    }

    console.log('starting worker...');

    const handler = hatchet.webhooks([workflow]);

    const server = createServer(handler.httpHandler({ secret }));

    await new Promise((resolve) => {
      server.listen(port, () => {
        resolve('');
      });
    });

    console.log('server started.');
    console.log('waiting for worker to be registered...');

    // wait for engine to pick up the webhook worker
    await sleep(30_000 + 10_000);

    console.log('webhook wait time complete.');

    console.log('pushing event...');

    await hatchet.event.push('user:create-webhook', {
      test: 'test',
    });

    await sleep(10000);

    console.log('invoked', invoked);

    // FIXME: add this back
    // expect(invoked).toEqual(2);

    await worker.stop();
  }, 60000);
});
