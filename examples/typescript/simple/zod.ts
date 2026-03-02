// > Declaring a Task
import * as z from 'zod';
import { hatchet } from '../hatchet-client';

const SimpleInputSchema = z.object({
  Message: z.string(),
});

// (optional) Define the input type for the workflow
export type SimpleInputWithZod = z.infer<typeof SimpleInputSchema>;

export const simpleWithZod = hatchet.task({
  name: 'simple-with-zod',
  retries: 3,
  fn: async (input: SimpleInputWithZod) => {
    return {
      TransformedMessage: input.Message.toLowerCase(),
    };
  },
  inputValidator: SimpleInputSchema,
});


// see ./worker.ts and ./run.ts for how to run the workflow
