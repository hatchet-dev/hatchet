import { nonRetryableWorkflow } from './workflow';

async function main() {
  const res = await nonRetryableWorkflow.runNoWait({});
  console.log(res);
}

if (require.main === module) {
  main();
}
