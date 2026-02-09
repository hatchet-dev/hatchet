import { simple } from './workflows/first-workflow';

async function main() {
  const res = await simple.run({
    message: 'hello',
  });

  console.log(res['first-task'].message);
  console.log(res['second-task'].message);
}

if (require.main === module) {
  main().catch(console.error).finally(() => process.exit(0));
}
