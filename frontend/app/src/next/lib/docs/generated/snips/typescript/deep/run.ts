import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { parent } from \'./workflow\';\n\nasync function main() {\n  const res = await parent.run({\n    Message: \'hello\',\n    N: 5,\n  });\n  console.log(res.parent.Sum);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n',
  'source': 'out/typescript/deep/run.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
