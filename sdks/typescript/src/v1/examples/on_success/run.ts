/* eslint-disable no-console */
import { onSuccessDag } from './workflow';

async function main() {
  try {
    const res2 = await onSuccessDag.run({});
    console.log(res2);
  } catch (e) {
    console.log('error', e);
  }
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
