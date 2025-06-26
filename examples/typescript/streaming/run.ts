import { RunEventType } from '@hatchet-dev/typescript-sdk-dev/typescript-sdk/clients/listeners/run-listener/child-listener-client';
import { streamingTask } from './workflow';

async function main() {
  // > Consume
  const ref = await streamingTask.runNoWait({});

  const stream = await ref.stream();

  for await (const event of stream) {
    if (event.type === RunEventType.STEP_RUN_EVENT_TYPE_STREAM) {
      console.log(event.payload);
    }
  }
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}
