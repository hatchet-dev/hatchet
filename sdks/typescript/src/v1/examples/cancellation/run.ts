/* eslint-disable no-console */
// > Running a Task with Results
import sleep from '@hatchet/util/sleep';
import { cancellationWorkflow } from './cancellation-workflow';
import { hatchet } from '../hatchet-client';
// ...
async function main() {
  const run = await cancellationWorkflow.runNoWait({});
  const run1 = await cancellationWorkflow.runNoWait({});

  await sleep(1000);

  await run.cancel();

  const res = await run.output;
  const res1 = await run1.output;

  console.log('canceled', res);
  console.log('completed', res1);

  await sleep(1000);

  await run.replay();

  const resReplay = await run.output;

  console.log(resReplay);

  const run2 = await cancellationWorkflow.runNoWait({}, { additionalMetadata: { test: 'abc' } });
  const run4 = await cancellationWorkflow.runNoWait({}, { additionalMetadata: { test: 'test' } });

  await sleep(1000);

  await hatchet.runs.cancel({
    filters: {
      since: new Date(Date.now() - 60 * 60),
      additionalMetadata: { test: 'test' },
    },
  });

  const res3 = await Promise.all([run2.output, run4.output]);
  console.log(res3);
  // !!
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}

