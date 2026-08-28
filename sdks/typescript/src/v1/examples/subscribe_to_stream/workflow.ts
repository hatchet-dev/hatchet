import sleep from '../../../util/sleep';
import { hatchet } from '../hatchet-client';

export const STREAM_CHUNKS = Array.from(
  { length: 20 },
  (_, i) => `c${String(i).padStart(2, '0')}|`
);

export const PRE_STREAM_SLEEP_MS = 2000;
export const INTER_CHUNK_MS = 50;

export const longStream = hatchet.task({
  name: 'subscribe-stream-long',
  fn: async (_, ctx) => {
    await sleep(PRE_STREAM_SLEEP_MS);
    for (const chunk of STREAM_CHUNKS) {
      await ctx.putStream(chunk);
      await sleep(INTER_CHUNK_MS);
    }
  },
});

// An onFailure handler makes the run a DAG. DAG QUEUED->RUNNING used to be
// published as workflow-run-finished, which hung up stream subscribers as
// soon as the run started.
export const dagStream = hatchet.task({
  name: 'subscribe-stream-dag',
  fn: async (_, ctx) => {
    await sleep(PRE_STREAM_SLEEP_MS);
    for (const chunk of STREAM_CHUNKS) {
      await ctx.putStream(chunk);
    }
  },
  onFailure: async () => {},
});
