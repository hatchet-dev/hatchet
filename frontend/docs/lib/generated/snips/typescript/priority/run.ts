import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { Priority } from \'@hatchet-dev/typescript-sdk/v1\';\nimport { priority } from \'./workflow\';\n\nasync function main() {\n  try {\n    console.log(\'running priority workflow\');\n\n    // > Run a Task with a Priority\n    const run = priority.run(new Date(Date.now() + 60 * 60 * 1000), { priority: Priority.HIGH });\n    \n\n    // > Schedule and cron\n    const scheduled = priority.schedule(\n      new Date(Date.now() + 60 * 60 * 1000),\n      {},\n      { priority: Priority.HIGH }\n    );\n    const delayed = priority.delay(60 * 60 * 1000, {}, { priority: Priority.HIGH });\n    const cron = priority.cron(\n      `daily-cron-${Math.random()}`,\n      \'0 0 * * *\',\n      {},\n      { priority: Priority.HIGH }\n    );\n    \n\n    const [scheduledResult, delayedResult] = await Promise.all([scheduled, delayed]);\n    console.log(\'scheduledResult\', scheduledResult);\n    console.log(\'delayedResult\', delayedResult);\n    \n  } catch (e) {\n    console.log(\'error\', e);\n  }\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n',
  'source': 'out/typescript/priority/run.ts',
  'blocks': {
    'run_a_task_with_a_priority': {
      'start': 9,
      'stop': 9
    },
    'schedule_and_cron': {
      'start': 12,
      'stop': 23
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
