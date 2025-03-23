import { durableSleep } from './workflow';

async function main() {
  const res = await durableSleep.run({
    N: 10,
  });

  // eslint-disable-next-line no-console
  console.log(res.value.Value);
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
