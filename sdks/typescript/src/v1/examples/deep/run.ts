import { parent } from './workflow';

async function main() {
  const res = await parent.run({
    Message: 'hello',
    N: 5,
  });

  // eslint-disable-next-line no-console
  console.log(res.parent.Sum);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
