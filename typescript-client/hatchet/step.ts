import * as z from 'zod';

export const CreateStepSchema = z.object({
  name: z.string(),
  parents: z.array(z.string()).optional(),
});

export type NextStep = { [key: string]: string };

export type Context = {};

export interface CreateStep<T> extends z.infer<typeof CreateStepSchema> {
  run: (input: T, ctx: Context) => NextStep;
}

const x: CreateStep<any> = {
  name: 'test',
  run: (input, ctx) => {
    return {};
  },
};
