import { hatchet } from '../client';
import { simple } from './workflow';

async function main() {
  const cron = await simple.cron('simple-daily', '0 0 * * *', {
    Message: 'hello',
  });

  // eslint-disable-next-line no-console
  console.log(cron.metadata.id);

  await hatchet.cron.delete(cron);
}

if (require.main === module) {
  main();
}
