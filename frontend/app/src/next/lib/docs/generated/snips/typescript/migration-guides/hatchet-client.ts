import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import HatchetClient from \'@hatchet-dev/typescript-sdk/sdk\';\n\nexport const hatchet = HatchetClient.init();\n',
  'source': 'out/typescript/migration-guides/hatchet-client.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
