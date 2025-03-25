import { hatchet } from '../client';

async function main() {
  const event = await hatchet.events.push('user:event', {
    Value: 'event',
  });
}

if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      // eslint-disable-next-line no-console
      console.error('Error:', error);
      process.exit(1);
    });
}
