import Hatchet from '../sdk';

const hatchet = Hatchet.init();

// Push a single event (example)
hatchet.events.push('user:create', {
  test: 'test',
});

// Example events to be pushed in bulk
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

// Bulk push the events and compare the keys
hatchet.events
  .bulkPush('user:create:bulk', events)
  .then((result) => {
    const returnedEvents = result.events;

    const keysMatch = returnedEvents.every((returnedEvent) => {
      const expectedKey = `user:create:bulk`;

      return returnedEvent.key === expectedKey;
    });

    if (keysMatch) {
      // eslint-disable-next-line no-console
      console.log('All keys match the original events.');
    } else {
      // eslint-disable-next-line no-console
      console.log('Mismatch found between original events and returned events.');
    }
  })
  .catch((error) => {
    // eslint-disable-next-line no-console
    console.error('Error during bulk push:', error);
  });
