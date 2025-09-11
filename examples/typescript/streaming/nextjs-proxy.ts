import { Readable } from 'stream';
import { hatchet } from '../hatchet-client';
import { streamingTask } from './workflow';

// > NextJS Proxy
export async function GET() {
  try {
    const ref = await streamingTask.runNoWait({});
    const workflowRunId = await ref.getWorkflowRunId();

    const stream = Readable.from(hatchet.runs.subscribeToStream(workflowRunId));

    // @ts-ignore
    return new Response(Readable.toWeb(stream), {
      headers: {
        'Content-Type': 'text/plain',
        'Cache-Control': 'no-cache',
        Connection: 'keep-alive',
      },
    });
  } catch (error) {
    return new Response('Internal Server Error', { status: 500 });
  }
}
