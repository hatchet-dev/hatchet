import { parent } from './workflow';

async function main() {
  const res = await parent.run({
    N: 10,
  });

  // eslint-disable-next-line no-console
  console.log(res.sum.Result);
}

if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error('Error:', error);
      process.exit(1);
    });
}
