// > Running a Task with Results
import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import { cancellation } from './workflow';
import { hatchet } from '../hatchet-client';
// ...
async function main() {
  const run = cancellation.runNoWait({});
  const run1 = cancellation.runNoWait({});

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

  const run2 = cancellation.runNoWait({}, { additionalMetadata: { test: 'abc' } });
  const run4 = cancellation.runNoWait({}, { additionalMetadata: { test: 'test' } });

  await sleep(1000);

  await hatchet.runs.cancel({
    filters: {
      since: new Date(Date.now() - 60 * 60),
      additionalMetadata: { test: 'test' },
    },
  });

  const res3 = await Promise.all([run2.output, run4.output]);
  console.log(res3);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
