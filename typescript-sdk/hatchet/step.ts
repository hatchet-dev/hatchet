import HatchetError from '@util/errors/hatchet-error';
import * as z from 'zod';

export const CreateStepSchema = z.object({
  name: z.string(),
  parents: z.array(z.string()).optional(),
});

export type NextStep = { [key: string]: string };

interface ContextData<T = unknown> {
  input: T;
  parents: Record<string, any>;
  triggered_by_event: string;
}

export class Context<T = unknown> {
  data: ContextData<T>;
  constructor(payload: string) {
    try {
      this.data = JSON.parse(JSON.parse(payload));
    } catch (e: any) {
      throw new HatchetError(`Could not parse payload: ${e.message}`);
    }
  }

  stepOutput(step: string): string {
    if (!this.data.parents) {
      throw new HatchetError('Step output not found');
    }
    if (!this.data.parents[step]) {
      throw new HatchetError(`Step output for '${step}' not found`);
    }
    return this.data.parents[step];
  }

  triggeredByEvent(): boolean {
    return this.data?.triggered_by_event === 'event';
  }

  workflowInput(): any {
    return this.data?.input || {};
  }
}

export interface CreateStep<T> extends z.infer<typeof CreateStepSchema> {
  run: (ctx: Context) => Promise<NextStep> | NextStep | void;
}
