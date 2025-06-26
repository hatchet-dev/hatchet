import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import sleep from '@hatchet-dev/typescript-sdk-dev/typescript-sdk/util/sleep';\nimport { hatchet } from '../hatchet-client';\n\n// > Streaming\nconst annaKarenina = `\nHappy families are all alike; every unhappy family is unhappy in its own way.\n\nEverything was in confusion in the Oblonskys' house. The wife had discovered that the husband was carrying on an intrigue with a French girl, who had been a governess in their family, and she had announced to her husband that she could not go on living in the same house with him.\n`;\n\nfunction* createChunks(content: string, n: number): Generator<string, void, unknown> {\n  for (let i = 0; i < content.length; i += n) {\n    yield content.slice(i, i + n);\n  }\n}\n\nexport const streamingTask = hatchet.task({\n  name: 'stream-example',\n  fn: async (_, ctx) => {\n    for (const chunk of createChunks(annaKarenina, 10)) {\n      ctx.putStream(chunk);\n      await sleep(200);\n    }\n  },\n});\n\n",
  source: 'out/typescript/streaming/workflow.ts',
  blocks: {
    streaming: {
      start: 5,
      stop: 26,
    },
  },
  highlights: {},
};

export default snippet;
