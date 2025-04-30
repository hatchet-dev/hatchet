import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'go',
  'content': 'package migration_guides\n\nimport (\n\tv1 \'github.com/hatchet-dev/hatchet/pkg/v1\'\n)\n\nfunc HatchetClient() (v1.HatchetClient, error) {\n\thatchet, err := v1.NewHatchetClient()\n\n\tif err != nil {\n\t\treturn nil, err\n\t}\n\n\treturn hatchet, nil\n}\n',
  'source': 'out/go/migration-guides/hatchet-client.go',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
