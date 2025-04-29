import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { dagWithConditions } from \'./workflow\';\n\nasync function main() {\n  const res = await dagWithConditions.run({});\n\n  console.log(res[\'first-task\'].Completed);\n  console.log(res[\'second-task\'].Completed);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n',
  'source': 'out/typescript/dag_match_condition/run.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
