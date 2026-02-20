import { hatchetWithMiddleware } from "./client";

type TaskInput = {
  message: string;
};

type TaskOutput = {
  message: string;
};

export const taskWithMiddleware = hatchetWithMiddleware.task<TaskInput, TaskOutput>({
  name: 'task-with-middleware',
  fn: (input, _ctx) => {
      console.log('task', input.message);   // string  (from TaskInput)
      console.log('task', input.first);     // number  (from GlobalInputType)
      console.log('task', input.second);    // number  (from GlobalInputType)
      console.log('task', input.requestId); // string  (from Middleware)
      return { message: input.message, extra: 1 };
  },
});

async function main() {
  const result = await taskWithMiddleware.run({
    message: 'hello',
    first: 1,
    second: 2,
  });

  console.log('result', result.message); // string  (from TaskOutput)
  console.log('result', result.extra);   // number  (from GlobalOutputType & MiddlewarePost)
}

if (require.main === module) {
  main();
}
// !!
