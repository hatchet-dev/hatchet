import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\nimport { simple } from \'./workflow\';\n\nasync function main() {\n  // > Create\n  const cron = await simple.cron(\'simple-daily\', \'0 0 * * *\', {\n    Message: \'hello\',\n  });\n\n  // it may be useful to save the cron id for later\n  const cronId = cron.metadata.id;\n  console.log(cron.metadata.id);\n\n  // > Delete\n  await hatchet.crons.delete(cronId);\n  \n\n  // > List\n  const crons = await hatchet.crons.list({\n    workflowId: simple.id,\n  });\n  console.log(crons);\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/simple/cron.ts',
  'blocks': {
    'create': {
      'start': 6,
      'stop': 11
    },
    'delete': {
      'start': 16,
      'stop': 16
    },
    'list': {
      'start': 19,
      'stop': 21
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
