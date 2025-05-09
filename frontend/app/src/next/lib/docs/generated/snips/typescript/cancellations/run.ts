import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "// > Running a Task with Results\nimport sleep from '@hatchet-dev/typescript-sdk/util/sleep';\nimport { cancellation } from './workflow';\nimport { hatchet } from '../hatchet-client';\n// ...\nasync function main() {\n  const run = await cancellation.runNoWait({});\n  const run1 = await cancellation.runNoWait({});\n\n  await sleep(1000);\n\n  await run.cancel();\n\n  const res = await run.output;\n  const res1 = await run1.output;\n\n  console.log('canceled', res);\n  console.log('completed', res1);\n\n  await sleep(1000);\n\n  await run.replay();\n\n  const resReplay = await run.output;\n\n  console.log(resReplay);\n\n  const run2 = await cancellation.runNoWait({}, { additionalMetadata: { test: 'abc' } });\n  const run4 = await cancellation.runNoWait({}, { additionalMetadata: { test: 'test' } });\n\n  await sleep(1000);\n\n  await hatchet.runs.cancel({\n    filters: {\n      since: new Date(Date.now() - 60 * 60),\n      additionalMetadata: { test: 'test' },\n    },\n  });\n\n  const res3 = await Promise.all([run2.output, run4.output]);\n  console.log(res3);\n}\n\nif (require.main === module) {\n  main()\n    .catch(console.error)\n    .finally(() => process.exit(0));\n}\n",
  source: 'out/typescript/cancellations/run.ts',
  blocks: {
    running_a_task_with_results: {
      start: 2,
      stop: 41,
    },
  },
  highlights: {},
}; // Then replace double quotes with single quotes

export default snippet;
