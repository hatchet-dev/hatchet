import { hatchetWithMiddleware } from "./client";

type TaskInput = {
  message: string;
};

type TaskOutput = {
  message: string;
};

// Note: for type safety with middleware, we need to explicitly specify the input and output types in the generic parameters
export const taskWithMiddleware = hatchetWithMiddleware.task<TaskInput, TaskOutput>({
  name: 'task-with-middleware',
  fn: (input, _ctx) => {
      console.log('task', input.data);      // number    (from first pre hook)
      console.log('task', input.requestId); // string    (from second pre hook)
      console.log('task', input.message);   // string    (from TaskWithMiddlewareInput)
      return { message: input.message };
  },
});
