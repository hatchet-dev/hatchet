import { taskWithMiddleware } from "./workflow";

async function main() {
  const result = await taskWithMiddleware.run({
      message: 'hello',
      first: 1,
      second: 2,
  });

  console.log('result', result.message); // string  (from TaskOutput)
}

if (require.main === module) {
  main();
}
// !!
