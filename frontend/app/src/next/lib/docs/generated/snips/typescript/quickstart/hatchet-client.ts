import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import Hatchet from '@hatchet-dev/typescript-sdk/sdk';\n\nexport const hatchet = Hatchet.init();\n",
  source: 'out/typescript/quickstart/hatchet-client.ts',
  blocks: {},
  highlights: {
    client: {
      lines: [3],
      strings: ['Client'],
    },
  },
};

export default snippet;
