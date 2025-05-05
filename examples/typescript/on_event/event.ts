import { hatchet } from '../hatchet-client';
import { Input } from './workflow';

async function main() {
  // > Pushing an Event
  const res = await hatchet.events.push<Input>('simple-event:create', {
    Message: 'hello',
  });

  console.log(res.eventId);
}

if (require.main === module) {
  main();
}
