import { hatchet } from './clients';
import { generate, execute } from './tasks'

export type OrchestrationTaskInput = {
  prompt: string;
};

export const orchestrate = hatchet.durableTask({
  name: 'orchestration-task',
  fn: async ({ prompt }: OrchestrationTaskInput) => {

    const { code } = await generate.run({
        prompt
    })

    // NOTE: uncomment this line for the demo of durable execution
    // throw new Error("🐛")

    const result = await execute.run({
        code
    })

    return {
        result
    };
  },
});
