/* eslint-disable no-console */
import { RunEventType } from '@hatchet-dev/typescript-sdk/clients/listeners/run-listener/child-listener-client';
import { streaming_task } from './workflow';

async function main() {
  const ref = await streaming_task.runNoWait({});

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
