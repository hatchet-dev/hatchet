import { Snippet } from '@/next/lib/docs/generated/snips/types';

const snippet: Snippet = {
  language: 'unknown',
  content:
    "import { Readable } from 'stream';\nimport { hatchet } from '../../../../../../hatchet-client';\nimport { streamingTask } from '../../../../../workflow';\n\nexport async function GET() {\n  try {\n    const ref = await streamingTask.runNoWait({});\n    const workflowRunId = await ref.getWorkflowRunId();\n\n    const stream = Readable.from(hatchet.runs.subscribeToStream(workflowRunId));\n\n    return new Response(Readable.toWeb(stream), {\n      headers: {\n        'Content-Type': 'text/plain',\n        'Cache-Control': 'no-cache',\n        Connection: 'keep-alive',\n      },\n    });\n  } catch (error) {\n    return new Response('Internal Server Error', { status: 500 });\n  }\n}\n",
  source: 'out/typescript/streaming/nextjs/src/app/api/stream/route.js',
  blocks: {},
  highlights: {},
};

export default snippet;
