import { hatchet } from '../hatchet-client';

async function main() {
  const event = await hatchet.events.push('user:update', {
    userId: '1234',
  });
}

if (require.main === module) {
  main()
    .then(() => process.exit(0))
    .catch((error) => {
      console.error('Error:', error);
      process.exit(1);
    });
}
