import HatchetError from '@util/errors/hatchet-error';
import * as z from 'zod';

export const CreateStepSchema = z.object({
  name: z.string(),
  parents: z.array(z.string()).optional(),
});

export type NextStep = { [key: string]: string };

export class Context<T = any> {
  data: T | any;
  constructor(payload: string) {
    try {
      this.data = JSON.parse(payload);
    } catch (e: any) {
      throw new HatchetError(`Could not parse payload: ${e.message}`);
    }
  }

  step_output(step: string): string {
    if (!this.data.parents) {
      throw new HatchetError('Step output not found');
    }
    if (!this.data.parents[step]) {
      throw new HatchetError(`Step output for '${step}' not found`);
    }
    return this.data.parents[step];
  }

  triggered_by_event(): boolean {
    return this.data?.triggered_by_event === 'event';
  }

  workflow_input(): any {
    return this.data?.input || {};
  }
}

export interface CreateStep<T> extends z.infer<typeof CreateStepSchema> {
  run: (input: T, ctx: Context) => NextStep;
}
