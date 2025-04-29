import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { HatchetClient } from \'@hatchet-dev/typescript-sdk\';\n\nclient 1 Client\nexport const hatchet = HatchetClient.init();\n',
  'source': 'out/typescript/quickstart/hatchet-client.ts',
  'blocks': {},
  'highlights': {
    'client': {
      'lines': [
        3
      ],
      'strings': [
        'Client'
      ]
    }
  }
};  // Then replace double quotes with single quotes

export default snippet;
