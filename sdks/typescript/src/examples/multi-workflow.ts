import Hatchet from '../sdk';

const hatchet: Hatchet = Hatchet.init();

async function main() {
  const worker = await hatchet.worker('test-worker2', 5);

  await worker.registerWorkflow({
    id: 'test1',
    description: 'desc',
    on: {
      event: 'test1',
    },
    steps: [
      {
        name: 'test1-step1',
        run: (ctx) => {
          console.log('executed step1 of test1!');
          return { step1: 'step1' };
        },
      },
    ],
  });

  await worker.registerWorkflow({
    id: 'test2',
    description: 'desc',
    on: {
      event: 'test2',
    },
    steps: [
      {
        name: 'test2-step1',
        run: (ctx) => {
          console.log('executed step1 of test2!');
          return { step1: 'step1' };
        },
      },
    ],
  });

  await worker.start();
}

main();
