import { durableEvent } from './workflow';

async function main() {
  const timeStart = Date.now();
  const res = await durableEvent.run({});
  const timeEnd = Date.now();
  // eslint-disable-next-line no-console
  console.log(`Time taken: ${timeEnd - timeStart}ms`);
}

if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      // eslint-disable-next-line no-console
      console.error('Error:', error);
      process.exit(1);
    });
}
