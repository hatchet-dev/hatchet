import { dag } from './workflow';

async function main() {
  const res = await dag.run({
    Message: 'hello world',
  });

  // eslint-disable-next-line no-console
  console.log(res.reverse.Transformed);
}

if (require.main === module) {
  main();
}
