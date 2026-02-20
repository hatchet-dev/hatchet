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
      console.log('task', input.message); // string  (from TaskInput)
      console.log('task', input.first);   // number  (from GlobalType)
      console.log('task', input.second);  // number  (from GlobalType)
      return { message: input.message };
  },
});

taskWithMiddleware.run({
  message: 'hello',
  first: 1,
  second: 2,
});
