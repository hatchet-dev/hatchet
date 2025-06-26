import sleep from '@hatchet-dev/typescript-sdk-dev/typescript-sdk/util/sleep';
import { hatchet } from '../hatchet-client';

const content = `
Happy families are all alike; every unhappy family is unhappy in its own way.

Everything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.
`

function* createChunks(content: string, n: number): Generator<string, void, unknown> {
    for (let i = 0; i < content.length; i += n) {
        yield content.slice(i, i + n);
    }
}

export const streaming_task = hatchet.task({
  name: 'stream-example',
  fn: async (_, ctx) => {
    for (const chunk of createChunks(content, 10)) {
      ctx.putStream(chunk);
      await sleep(200);
    }
  },
});


