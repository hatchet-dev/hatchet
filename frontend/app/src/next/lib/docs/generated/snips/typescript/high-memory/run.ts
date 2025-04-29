import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { parent } from \'./workflow-with-child\';\n\nasync function main() {\n  // ❓ Running a Task\n  const res = await parent.run({\n    Message: \'HeLlO WoRlD\',\n  });\n\n  // 👀 Access the results of the Task\n  console.log(res.TransformedMessage);\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/high-memory/run.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
