import { hatchet } from '../hatchet-client';
import { Input } from './workflow';

async function main() {
  const res = await hatchet.events.push<Input>(SIMPLE_EVENT, {
    Message: 'hello',
  });
  // !!

  // eslint-disable-next-line no-console
  console.log(res.eventId);
}

if (require.main === module) {
  main();
}
