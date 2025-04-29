import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { parent } from \'./workflow\';\n\nasync function main() {\n  const res = await parent.run({\n    N: 10,\n  });\n  console.log(res.Result);\n}\n\nif (require.main === module) {\n  main()\n    .then(() => process.exit(0))\n    .catch((error) => {\n      console.error(\'Error:\', error);\n      process.exit(1);\n    });\n}\n',
  'source': 'out/typescript/child_workflows/run.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
