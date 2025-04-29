import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  'language': 'typescript ',
  'content': 'import { NonRetryableError } from \'@hatchet-dev/typescript-sdk/v1/task\';\nimport { hatchet } from \'../hatchet-client\';\n\nexport const nonRetryableWorkflow = hatchet.workflow({\n  name: \'no-retry-workflow\',\n});\n\n// > Non-retrying task\nconst shouldNotRetry = nonRetryableWorkflow.task({\n  name: \'should-not-retry\',\n  fn: () => {\n    throw new NonRetryableError(\'This task should not retry\');\n  },\n  retries: 1,\n});\n\n// Create a task that should retry\nconst shouldRetryWrongErrorType = nonRetryableWorkflow.task({\n  name: \'should-retry-wrong-error-type\',\n  fn: () => {\n    throw new Error(\'This task should not retry\');\n  },\n  retries: 1,\n});\n\nconst shouldNotRetrySuccessfulTask = nonRetryableWorkflow.task({\n  name: \'should-not-retry-successful-task\',\n  fn: () => {},\n});\n',
  'source': 'out/typescript/non_retryable/workflow.ts',
  'blocks': {
    'non_retrying_task': {
      'start': 9,
      'stop': 15
    }
  },
  'highlights': {}
};  // Then replace double quotes with single quotes

export default snippet;
