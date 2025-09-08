import { hatchet } from '../hatchet-client';
import { Input } from './workflow';

async function main() {
  // > Pushing an Event
  const res = await hatchet.events.push<Input>('simple-event:create', {
    Message: 'hello',
    ShouldSkip: false,
  });
  // !!

  // > Push an Event with Metadata
  const withMetadata = await hatchet.events.push(
    'user:create',
    {
      test: 'test',
    },
    {
      additionalMetadata: {
        source: 'api', // Arbitrary key-value pair
      },
    }
  );
  // !!

  // eslint-disable-next-line no-console
  console.log(res.eventId);
}

if (require.main === module) {
  main();
}
