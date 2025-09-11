import { nonRetryableWorkflow } from './workflow';

async function main() {
  const res = await nonRetryableWorkflow.runNoWait({});

  // eslint-disable-next-line no-console
  console.log(res);
}

if (require.main === module) {
  main();
}
