import type { DurableEvictionManager } from '@hatchet-dev/typescript-sdk/v1/client/worker/eviction/eviction-manager';
import { hatchet } from '../hatchet-client';
import { evictableSleepForGracefulTermination } from './workflow';

export const DURABLE_WAIT_MARKED = '[e2e] durable wait marked';

// Worker for the graceful termination e2e test. The test must not terminate
// this worker before the SDK has recorded the run as waiting, and nothing
// observable outside the process marks that moment: the engine writes the
// WAIT_FOR entry before it acks, and the SDK only calls markWaiting once the
// ack arrives. The eviction manager is created lazily on the worker's
// internal state, so it is wrapped on assignment to print a line the test can
// wait for.
async function main() {
  const worker = await hatchet.worker('graceful-eviction-worker', {
    workflows: [evictableSleepForGracefulTermination],
  });

  let manager: DurableEvictionManager | undefined;
  Object.defineProperty(worker._internal, 'evictionManager', {
    configurable: true,
    enumerable: true,
    get: () => manager,
    set: (next: DurableEvictionManager | undefined) => {
      manager = next;
      if (!next) return;
      const markWaiting = next.markWaiting.bind(next);
      next.markWaiting = (key, waitKind, resourceId) => {
        markWaiting(key, waitKind, resourceId);
        console.log(`${DURABLE_WAIT_MARKED} key=${key} kind=${waitKind} resource=${resourceId}`);
      };
    },
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
