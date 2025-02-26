import Hatchet from '../sdk';

const hatchet = Hatchet.init();

async function main() {
  // Generate a random stream key to use to track all
  // stream events for this workflow run.
  const streamKey = 'streamKey';
  const streamVal = `sk-${Math.random().toString(36).substring(7)}`;

  // Specify the stream key as additional metadata
  // when running the workflow.

  // This key gets propagated to all child workflows
  // and can have an arbitrary property name.
  await hatchet.admin.runWorkflow(
    'parent-workflow',
    {},
    { additionalMetadata: { [streamKey]: streamVal } }
  );

  // Stream all events for the additional meta key value
  const stream = await hatchet.listener.streamByAdditionalMeta(streamKey, streamVal);

  for await (const event of stream) {
    console.log('event received', event);
  }
}

main();
