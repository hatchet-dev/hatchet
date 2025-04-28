import { hatchet } from '../hatchet-client';
import { Input } from './workflow';

async function main() {
  // > Pushing an Event
  const res = await hatchet.event.push<Input>('simple-event:create', {
    Message: 'hello',
  });
  // !!

  // eslint-disable-next-line no-console
  console.log(res.eventId);
}

if (require.main === module) {
  main();
}
