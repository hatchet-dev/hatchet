import { simple } from './workflow';

async function main() {
  const res = await simple.run({
    Message: 'hello',
  });

  // eslint-disable-next-line no-console
  console.log(res.step2);
}

if (require.main === module) {
  main();
}
