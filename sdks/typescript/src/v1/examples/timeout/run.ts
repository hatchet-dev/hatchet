/* eslint-disable no-console */
import { refreshTimeoutTask, timeoutTask } from './workflow';

async function main() {
  try {
    await timeoutTask.run({ Message: 'hello' });
  } catch (e) {
    console.log('timeoutTask failed as expected', e);
  }

  const res = await refreshTimeoutTask.run({ Message: 'hello' });
  console.log(res);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}

