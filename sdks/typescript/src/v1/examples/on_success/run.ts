/* eslint-disable no-console */
import { onSuccess, onSuccessDag } from './workflow';

async function main() {
  try {
    const res = await onSuccess.run({});
    console.log(res);

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
