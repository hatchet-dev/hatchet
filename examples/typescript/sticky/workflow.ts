import { StickyStrategy } from '@hatchet-dev/typescript-sdk/protoc/workflows';
import { hatchet } from '../hatchet-client';
import { child } from '../child_workflows/workflow';

// > Sticky Task
export const sticky = hatchet.task({
  name: 'sticky',
  retries: 3,
  sticky: StickyStrategy.SOFT,
  fn: async (_, ctx) => {
    // specify a child workflow to run on the same worker
    const result = await ctx.runChild(
      child,
      {
        N: 1,
      },
      { sticky: true }
    );

    return {
      result,
    };
  },
});
