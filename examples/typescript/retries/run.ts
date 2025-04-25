
import { retries } from './workflow';

async function main() {
  try {
    const res = await retries.run({});
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
