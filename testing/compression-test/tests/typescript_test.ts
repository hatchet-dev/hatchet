import Hatchet from './dist/index';

// Create large payload (100KB)
const createLargePayload = (): Record<string, string> => {
  const payload: Record<string, string> = {};
  const chunk = 'a'.repeat(1000); // 1KB chunk
  for (let i = 0; i < 100; i++) {
    payload[`chunk_${i}`] = chunk;
  }
  return payload;
};

interface TestEvent {
  id: number;
  createdAt: string;
  payload: Record<string, string>;
}

async function main() {
  const hatchet = Hatchet.init({
    namespace: process.env.HATCHET_CLIENT_NAMESPACE || 'compression-test',
  });

  const worker = await hatchet.worker('compression-test-worker', {
    slots: 100,
  });

  // Get compression state from environment (default to 'enabled')
  const compressionState = process.env.COMPRESSION_STATE || 'enabled';
  const workflowId = `${compressionState}-typescript`;

  // Register workflow
  await worker.registerWorkflow({
    id: workflowId,
    description: 'Test workflow for compression testing',
    on: {
      event: 'compression-test:event',
    },
    steps: [
      {
        name: 'step1',
        run: async (ctx) => {
          const input = ctx.workflowInput() as TestEvent;
          console.log(`Processing event ${input.id}`);
          return {
            processed: true,
            eventId: input.id,
            timestamp: new Date().toISOString(),
          };
        },
      },
    ],
  });

  // Get number of events from environment variable
  const totalEvents = parseInt(process.env.TEST_EVENTS_COUNT || '10', 10);
  const eventsPerSecond = 10;
  const interval = 1000 / eventsPerSecond; // 100ms between events
  const duration = Math.max(1000, (totalEvents / eventsPerSecond) * 1000); // Calculate duration from events

  // Start worker in background (don't await - it's blocking)
  console.log('Starting worker...');
  const workerPromise = worker.start().catch((error) => {
    console.error('Worker error:', error);
  });

  // Wait for worker to register
  await new Promise((resolve) => setTimeout(resolve, 5000));

  console.log(`Emitting ${totalEvents} events over ${duration / 1000} seconds...`);

  const largePayload = createLargePayload();
  let eventId = 0;

  const emitEvents = async () => {
    while (eventId < totalEvents) {
      const event: TestEvent = {
        id: eventId++,
        createdAt: new Date().toISOString(),
        payload: largePayload,
      };

      try {
        await hatchet.events.push('compression-test:event', event);
      } catch (error) {
        console.error(`Error pushing event ${eventId}:`, error);
      }

      // Wait for next interval
      await new Promise((resolve) => setTimeout(resolve, interval));
    }

    console.log(`Finished emitting ${eventId} events`);
  };

  // Start emitting events and wait for completion
  await emitEvents();

  // Wait additional time for events to be processed
  // Add buffer for processing time (events take time to execute)
  const processingBuffer = 10000; // 10 seconds buffer for processing
  const waitTime = duration + processingBuffer;
  console.log(`Waiting ${waitTime / 1000} seconds for events to be processed...`);
  await new Promise((resolve) => setTimeout(resolve, waitTime));

  console.log('Test complete, stopping worker...');
  try {
    // Stop worker with a timeout to prevent hanging
    await Promise.race([
      worker.stop(),
      new Promise((_, reject) =>
        setTimeout(() => reject(new Error('Worker stop timeout')), 10000)
      )
    ]);
  } catch (error) {
    console.error('Error stopping worker:', error);
    // Force exit if stop hangs
  }
  console.log('Worker stopped, exiting...');
  process.exit(0);
}

main().catch((error) => {
  console.error('Test failed:', error);
  process.exit(1);
});
