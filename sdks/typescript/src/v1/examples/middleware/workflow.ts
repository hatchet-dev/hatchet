import { hatchetWithMiddleware } from './client';

type TaskInput = {
  message: string;
};

type TaskOutput = {
  message: string;
};

export const taskWithMiddleware = hatchetWithMiddleware.task<TaskInput, TaskOutput>({
  name: 'task-with-middleware',
  fn: (input, ctx) => {
    ctx.logger.info('task', input.message); // string  (from TaskInput)
    ctx.logger.info('task', input.first); // number  (from GlobalInputType)
    ctx.logger.info('task', input.second); // number  (from GlobalInputType)
    ctx.logger.info('task', input.dependency); // string  (from Pre Middleware)
    return {
      message: input.message,
      extra: 1,
    };
  },
});
// !!
