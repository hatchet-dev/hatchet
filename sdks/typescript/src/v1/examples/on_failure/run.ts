/* eslint-disable no-console */
import { alwaysFail } from './workflow';

async function main() {
  try {
    const res = await alwaysFail.run({});
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
