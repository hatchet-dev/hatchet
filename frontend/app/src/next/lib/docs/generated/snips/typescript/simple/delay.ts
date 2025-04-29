import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\nimport { simple } from \'./workflow\';\n\nasync function main() {\n  const tomorrow = 24 * 60 * 60; // 1 day\n  const scheduled = await simple.delay(tomorrow, {\n    Message: \'hello\',\n  });\n  console.log(scheduled.metadata.id);\n\n  await hatchet.schedules.delete(scheduled);\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/simple/delay.ts',
  'blocks': {},
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
