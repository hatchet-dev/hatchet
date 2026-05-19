import { hatchet } from '../hatchet-client';
import {
  evictableSleep,
  evictableWaitForEvent,
  evictableChildSpawn,
  evictableChildBulkSpawn,
  multipleEviction,
  nonEvictableSleep,
  childTask,
  bulkChildTask,
  evictableSleepForGracefulTermination,
} from './workflow';

async function main() {
  const worker = await hatchet.worker('eviction-worker', {
    workflows: [
      evictableSleep,
      evictableWaitForEvent,
      evictableChildSpawn,
      evictableChildBulkSpawn,
      multipleEviction,
      nonEvictableSleep,
      childTask,
      bulkChildTask,
      evictableSleepForGracefulTermination,
    ],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
