/**
 * Dedicated worker for capacity-eviction e2e tests.
 *
 * Runs with durableSlots=1 so that a single waiting durable task triggers
 * capacity pressure and gets evicted (even with ttl=undefined).
 */
import { hatchet } from '../hatchet-client';
import { capacityEvictableSleep } from './workflow';

async function main() {
  const worker = await hatchet.worker('capacity-eviction-worker', {
    durableSlots: 1,
    workflows: [capacityEvictableSleep],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
