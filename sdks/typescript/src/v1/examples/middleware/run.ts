import { taskWithMiddleware } from "./workflow";

async function main() {
  const result = await taskWithMiddleware.run({
      message: 'hello',
  });

  console.log('result', result.extra);   // number  (from post middleware)
  console.log('result', result.message); // string  (from TaskWithMiddlewareOutput)
}

if (require.main === module) {
  main();
}
// !!
