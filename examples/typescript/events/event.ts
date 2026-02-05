import { hatchet } from '../hatchet-client';
import { Input } from './workflow';

async function main() {
  // > Pushing an Event
  const res = await hatchet.events.push<Input>('simple-event:create', {
    Message: 'hello',
    ShouldSkip: false,
  });

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

  // > Bulk push events
  const events = [
    {
      payload: { test: 'test1' },
      additionalMetadata: { user_id: 'user1', source: 'test' },
    },
    {
      payload: { test: 'test2' },
      additionalMetadata: { user_id: 'user2', source: 'test' },
    },
    {
      payload: { test: 'test3' },
      additionalMetadata: { user_id: 'user3', source: 'test' },
    },
  ];

  await hatchet.events.bulkPush('user:create', events);

  console.log(res.eventId);
}

if (require.main === module) {
  main();
}

