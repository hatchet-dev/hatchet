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
      console.log('task', input.dependency); // string  (from Pre Middleware)
      return {
        message: input.message,
        extra: 1,
      };
  },
});
