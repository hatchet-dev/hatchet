import { randomUUID } from 'crypto';
import { hatchet } from '../hatchet-client';
import { progressTask } from './workflow';

const decode = (payload: Uint8Array) => new TextDecoder().decode(payload);

export async function consume() {
  // > Consume
  const topic = `progress-${randomUUID()}`;
  await progressTask.runNoWait({ topic });

  for await (const event of hatchet.streams.subscribe(topic)) {
    const { step, total } = JSON.parse(decode(event.payload));
    console.log(`step ${step} of ${total}`);

    // the stream outlives the run, so the reader decides when it's done
    if (step === total) {
      break;
    }
  }
}

export async function publishFromAnywhere() {
  // > Publish from anywhere
  await hatchet.streams.publish(
    'orders',
    JSON.stringify({ orderId: 'ord_123', status: 'shipped' })
  );
}

export async function resumeFromCursor(topic: string) {
  // > Resume from a cursor
  let cursor: string | undefined;

  for await (const event of hatchet.streams.subscribe(topic)) {
    console.log(decode(event.payload));
    ({ cursor } = event);
    break;
  }

  // picks up right after the last event read, even from another process
  for await (const event of hatchet.streams.subscribe(topic, { cursor })) {
    console.log(decode(event.payload));
  }
}

export async function stopWithSignal(topic: string) {
  // > Stop reading
  const controller = new AbortController();
  setTimeout(() => controller.abort(), 10_000);

  for await (const event of hatchet.streams.subscribe(topic, { signal: controller.signal })) {
    console.log(decode(event.payload));
  }
}

export async function namespaces() {
  // > Namespaces
  await hatchet.streams.publish('updates', 'hello', { namespace: 'customer-a' });

  for await (const event of hatchet.streams.subscribe('updates', { namespace: 'customer-a' })) {
    console.log(decode(event.payload));
    break;
  }
}

export async function resetTopic() {
  // > Reset a topic
  await hatchet.streams.reset('orders');

  // starts after the reset; readers that were already iterating aren't affected
  for await (const event of hatchet.streams.subscribe('orders')) {
    console.log(decode(event.payload));
  }
}

export async function resetToCursor() {
  // > Reset to a cursor
  let cursor: string | undefined;

  for await (const event of hatchet.streams.subscribe('orders')) {
    ({ cursor } = event);
    break;
  }

  await hatchet.streams.reset('orders', { cursor });

  // starts right after the event the cursor was read from
  for await (const event of hatchet.streams.subscribe('orders')) {
    console.log(decode(event.payload));
  }
}

async function main() {
  await consume();
  await publishFromAnywhere();
  await stopWithSignal('orders');
  await namespaces();
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => {
      process.exit(0);
    });
}
