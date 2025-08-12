import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { hatchet } from '../hatchet-client';\nimport { simpleConcurrency } from './workflow';\n\nfunction generateRandomString(length: number): string {\n  const characters = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789';\n  let result = '';\n  for (let i = 0; i < length; i++) {\n    result += characters.charAt(Math.floor(Math.random() * characters.length));\n  }\n  return result;\n}\n\nasync function main() {\n  const groupCount = 2;\n  const runsPerGroup = 20_000;\n  const BATCH_SIZE = 400;\n\n  const workflowRuns = [];\n  for (let i = 0; i < groupCount; i++) {\n    for (let j = 0; j < runsPerGroup; j++) {\n      workflowRuns.push({\n        workflowName: simpleConcurrency.definition.name,\n        input: {\n          Message: generateRandomString(10),\n          GroupKey: `group-${i}`,\n        },\n      });\n    }\n  }\n\n  // Shuffle the workflow runs array\n  for (let i = workflowRuns.length - 1; i > 0; i--) {\n    const j = Math.floor(Math.random() * (i + 1));\n    [workflowRuns[i], workflowRuns[j]] = [workflowRuns[j], workflowRuns[i]];\n  }\n\n  // Process workflows in batches\n  for (let i = 0; i < workflowRuns.length; i += BATCH_SIZE) {\n    const batch = workflowRuns.slice(i, i + BATCH_SIZE);\n    await hatchet.admin.runWorkflows(batch);\n  }\n}\n\nif (require.main === module) {\n  main().then(() => process.exit(0));\n}\n",
  source: 'out/typescript/concurrency-rr/load.ts',
  blocks: {},
  highlights: {},
};

export default snippet;
