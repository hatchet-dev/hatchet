import { Snippet } from '@/lib/generated/snips/types';

const snippet: Snippet = {
  "language": "typescript ",
  "content": "// > Running a Task with Results\nimport { cancellation } from './workflow';\n// ...\nasync function main() {\n  // 👀 Run the workflow with results\n  const res = await cancellation.run({});\n\n  // 👀 Access the results of the workflow\n  console.log(res.Completed);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
  "source": "out/typescript/timeouts/run.ts",
  "blocks": {
    "running_a_task_with_results": {
      "start": 2,
      "stop": 9
    }
  },
  "highlights": {}
};

export default snippet;
