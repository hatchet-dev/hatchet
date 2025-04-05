/* eslint-disable no-console */
import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  // ❓ Running a Task
  const res = await simple.runNoWait(
    {
      Message: 'HeLlO WoRlD',
    },
    {
      additionalMetadata: {
        runId: '123',
      },
    }
  );
  const res2 = await simple.runNoWait(
    {
      Message: 'HeLlO WoRlD',
    },
    {
      additionalMetadata: {
        runId: 'abc',
      },
    }
  );

  const runs = await hatchet.runs.list({
    additionalMetadata: ['runId:123'],
  });

  console.log(runs);
}
if (require.main === module) {
  main();
}
