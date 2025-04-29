import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { dag } from \'./workflow\';\n\nasync function main() {\n  const res = await dag.run({\n    Message: \'hello world\',\n  });\n  console.log(res.reverse.Transformed);\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/dag/run.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
