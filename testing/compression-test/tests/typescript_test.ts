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

  // Register workflow
  await worker.registerWorkflow({
    id: 'compression-test-workflow',
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

  // Start worker
  console.log('Starting worker...');
  await worker.start();

  // Wait for worker to register
  await new Promise((resolve) => setTimeout(resolve, 5000));

  // Get number of events from environment variable
  const totalEvents = parseInt(process.env.TEST_EVENTS_COUNT || '10', 10);
  const eventsPerSecond = 10;
  const interval = 1000 / eventsPerSecond; // 100ms between events
  const duration = Math.max(1000, (totalEvents / eventsPerSecond) * 1000); // Calculate duration from events

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

  // Start emitting events
  emitEvents().catch(console.error);

  // Wait for test duration + buffer
  const waitTime = duration + 10000;
  await new Promise((resolve) => setTimeout(resolve, waitTime));

  console.log('Test complete, stopping worker...');
  await worker.stop();
  process.exit(0);
}

main().catch((error) => {
  console.error('Test failed:', error);
  process.exit(1);
});

