import Hatchet from '../sdk';

const hatchet = Hatchet.init();

async function main() {
  const workflowRun = hatchet.admin.runWorkflow('simple-workflow', {});
  const stream = await workflowRun.stream();

  for await (const event of stream) {
    console.log('event received', event);
  }
}

main();
