import { multiConcurrency } from './workflow';

async function main() {
  const res = await multiConcurrency.run([
    {
      Message: 'Hello World',
      GroupKey: 'A',
    },
    {
      Message: 'Goodbye Moon',
      GroupKey: 'A',
    },
    {
      Message: 'Hello World B',
      GroupKey: 'B',
    },
  ]);

  // eslint-disable-next-line no-console
  console.log(res[0]['to-lower'].TransformedMessage);
  // eslint-disable-next-line no-console
  console.log(res[1]['to-lower'].TransformedMessage);
  // eslint-disable-next-line no-console
  console.log(res[2]['to-lower'].TransformedMessage);
}

if (require.main === module) {
  main().then(() => process.exit(0));
}
