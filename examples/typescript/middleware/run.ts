import { taskWithMiddleware } from './workflow';

async function main() {
  // > Running a task with middleware
  const result = await taskWithMiddleware.run({
    message: 'hello', // string  (from TaskInput)
    first: 1, // number  (from GlobalInputType)
    second: 2, // number  (from GlobalInputType)
  });

  console.log('result', result.message); // string  (from TaskOutput)
  console.log('result', result.extra); // number  (from GlobalOutputType)
  console.log('result', result.additionalData); // number  (from Post Middleware)
}

if (require.main === module) {
  main();
}
