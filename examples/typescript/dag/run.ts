import { dag } from './workflow';

async function main() {
  const res = await dag.run({
    Message: 'hello world',
  });

  console.log(res.reverse.Transformed);
}

if (require.main === module) {
  main();
}
