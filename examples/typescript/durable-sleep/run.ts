import { durableSleep } from './workflow';

async function main() {
  const timeStart = Date.now();
  const res = await durableSleep.run({});
  const timeEnd = Date.now();
  console.log(`Time taken: ${timeEnd - timeStart}ms`);
}

if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error('Error:', error);
      process.exit(1);
    });
}
