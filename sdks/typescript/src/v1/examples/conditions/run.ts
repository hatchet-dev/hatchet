import { dagWithConditions } from './workflow';

async function main() {
  const res = await dagWithConditions.run({});

  console.log(res['first-task'].Completed);
  console.log(res['second-task'].Completed);
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
