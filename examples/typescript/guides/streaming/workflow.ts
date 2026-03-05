import { ConcurrencyLimitStrategy } from '@hatchet-dev/typescript-sdk/protoc/v1/workflows';
import { hatchet } from '../../hatchet-client';

// > Step 01 Define Streaming Task
export const streamTask = hatchet.task({
  name: 'stream-example',
  concurrency: {
    expression: "'constant'",
    maxRuns: 1,
    limitStrategy: ConcurrencyLimitStrategy.CANCEL_IN_PROGRESS,
  },
  fn: async (_, ctx) => {
    for (let i = 0; i < 5; i++) {
      ctx.putStream(`chunk-${i}`);
      await new Promise((r) => setTimeout(r, 500));
    }
    return { status: 'done' };
  },
});

// > Step 02 Emit Chunks
async function emitChunks(ctx: { putStream: (chunk: string) => void }) {
  for (let i = 0; i < 5; i++) {
    ctx.putStream(`chunk-${i}`);
    await new Promise((r) => setTimeout(r, 500));
  }
}
