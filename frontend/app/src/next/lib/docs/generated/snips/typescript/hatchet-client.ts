import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { HatchetClient } from '@hatchet-dev/typescript-sdk/v1';\n\nexport const hatchet = HatchetClient.init();\n",
  source: 'out/typescript/hatchet-client.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
