import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { hatchet } from \'../hatchet-client\';\nimport { simple } from \'./workflow\';\n\nasync function main() {\n  // > Create a Scheduled Run\n\n  const runAt = new Date(new Date().setHours(12, 0, 0, 0) + 24 * 60 * 60 * 1000);\n\n  const scheduled = await simple.schedule(runAt, {\n    Message: \'hello\',\n  });\n\n  // 👀 Get the scheduled run ID of the workflow\n  // it may be helpful to store the scheduled run ID of the workflow\n  // in a database or other persistent storage for later use\n  const scheduledRunId = scheduled.metadata.id;\n  console.log(scheduledRunId);\n\n  // > Delete a Scheduled Run\n  await hatchet.schedules.delete(scheduled);\n\n  // > List Scheduled Runs\n  const scheduledRuns = await hatchet.schedules.list({\n    workflowId: simple.id,\n  });\n  console.log(scheduledRuns);\n}\n\nif (require.main === module) {\n  main();\n}\n',
  'source': 'out/typescript/simple/schedule.ts',
  'blocks': {
    'create_a_scheduled_run': {
      'start': 6,
      'stop': 17
    },
    'delete_a_scheduled_run': {
      'start': 20,
      'stop': 20
    },
    'list_scheduled_runs': {
      'start': 23,
      'stop': 26
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
