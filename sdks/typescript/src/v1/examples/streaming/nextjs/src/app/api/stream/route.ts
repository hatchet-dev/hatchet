import { hatchet } from '../../../../../../hatchet-client';
import { streamingTask } from '../../../../../workflow';

export async function GET() {
  try {
    const ref = await streamingTask.runNoWait({});
    const workflowRunId = await ref.getWorkflowRunId();

    const encoder = new TextEncoder();
    const stream = new ReadableStream({
      async start(controller) {
        try {
          let chunkCount = 0;
          for await (const content of hatchet.runs.subscribeToStream(workflowRunId)) {
            console.log(`Received chunk ${chunkCount + 1}:`, content);
            chunkCount += 1;

            controller.enqueue(encoder.encode(content));
          }

          console.log(`Stream completed with ${chunkCount} chunks.`);

          controller.close();
        } catch (error) {
          controller.error(error);
        }
      },
    });

    return new Response(stream, {
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
