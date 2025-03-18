import { hatchet } from '../client';
import { SIMPLE_EVENT } from './workflow';

async function main() {
  const res = await hatchet.event.push(SIMPLE_EVENT, {
    Message: 'hello',
  });

  // eslint-disable-next-line no-console
  console.log(res.eventId);
}

if (require.main === module) {
  main();
}
