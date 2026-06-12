import { hatchet } from '../hatchet-client';
import { lookupCustomer, checkOrderStatus, createTicket } from './tools';

async function main() {
  const worker = await hatchet.worker('support-tools-worker', {
    workflows: [lookupCustomer, checkOrderStatus, createTicket],
  });

  await worker.start();
}

if (require.main === module) {
  main();
}
