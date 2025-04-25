import { firstTask } from './workflows/first-task';

async function main() {
  const res = await firstTask.run({
    Message: 'Hello World!',
  });

  console.log(
    'Finished running task, and got the transformed message! The transformed message is:',
    res.TransformedMessage
  );
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
