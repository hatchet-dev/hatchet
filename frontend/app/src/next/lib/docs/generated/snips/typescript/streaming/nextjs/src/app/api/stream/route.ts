import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'typescript ',
  content:
    "import { Readable } from 'stream';\nimport { hatchet } from '../../../../../../hatchet-client';\nimport { streamingTask } from '../../../../../workflow';\n\n// > NextJS Proxy\nexport async function GET() {\n  try {\n    const ref = await streamingTask.runNoWait({});\n    const workflowRunId = await ref.getWorkflowRunId();\n\n    const stream = Readable.from(hatchet.runs.subscribeToStream(workflowRunId));\n\n    // @ts-ignore\n    return new Response(Readable.toWeb(stream), {\n      headers: {\n        'Content-Type': 'text/plain',\n        'Cache-Control': 'no-cache',\n        Connection: 'keep-alive',\n      },\n    });\n  } catch (error) {\n    return new Response('Internal Server Error', { status: 500 });\n  }\n}\n",
  source: 'out/typescript/streaming/nextjs/src/app/api/stream/route.ts',
  blocks: {
    nextjs_proxy: {
      start: 6,
      stop: 24,
    },
  },
  highlights: {},
};

export default snippet;
