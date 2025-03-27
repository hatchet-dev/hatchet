/* eslint-disable no-console */
import { rateLimitWorkflow } from './workflow';

async function main() {
  try {
    const res = await rateLimitWorkflow.run({ userId: 'abc' });
    console.log(res);
  } catch (e) {
    console.log('error', e);
  }
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
