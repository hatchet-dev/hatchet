/* eslint-disable no-console */
import { streamingTask } from './workflow';
import { hatchet } from '../hatchet-client';

async function main() {
  // > Consume
  const ref = await streamingTask.runNoWait({});
  const id = await ref.getWorkflowRunId();

  for await (const content of hatchet.runs.subscribeToStream(id)) {
    process.stdout.write(content);
  }
  // !!
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}
