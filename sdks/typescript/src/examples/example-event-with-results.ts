import Hatchet from '../sdk';

async function main() {
  const hatchet = Hatchet.init();

  const ref = await hatchet.admin.runWorkflow('simple-workflow', {
    test: 'test',
  });

  const listener = await hatchet.v0.listener.stream(await ref.getWorkflowRunId());

  console.log('listening for events');
  for await (const event of listener) {
    console.log('event received', event);
  }
  console.log('done listening for events');
}

main();
