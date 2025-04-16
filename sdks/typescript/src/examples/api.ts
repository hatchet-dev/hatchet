import Hatchet, { V0Context } from '..';
import { CreateWorkflowVersionOpts } from '../protoc/workflows';

type CustomUserData = {
  example: string;
};

const opts: CreateWorkflowVersionOpts = {
  name: 'api-workflow',
  description: 'My workflow',
  version: '',
  eventTriggers: [],
  cronTriggers: [],
  scheduledTriggers: [],
  concurrency: undefined,
  jobs: [
    {
      name: 'my-job',
      description: 'Job description',
      steps: [
        {
          retries: 0,
          readableId: 'custom-step',
          action: `slack:example`,
          timeout: '60s',
          inputs: '{}',
          parents: [],
          workerLabels: {},
          userData: `{
            "example": "value"
          }`,
          rateLimits: [],
        },
      ],
    },
  ],
};

type StepOneInput = {
  key: string;
};

async function main() {
  const hatchet = Hatchet.init();

  const { admin } = hatchet;

  await admin.putWorkflow(opts);

  const worker = await hatchet.worker('example-worker');

  worker.nonDurable.registerAction(
    'slack:example',
    async (ctx: V0Context<StepOneInput, CustomUserData>) => {
      const setData = ctx.userData();
      console.log('executed step1!', setData);
      return { step1: 'step1' };
    }
  );

  await hatchet.admin.runWorkflow('api-workflow', {});

  worker.start();
}

main();
