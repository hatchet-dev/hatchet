import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  // ❓ Create
  const cron = await simple.cron('simple-daily', '0 0 * * *', {
    Message: 'hello',
  });

  // it may be useful to save the cron id for later
  const cronId = cron.metadata.id;
  // !!

  // eslint-disable-next-line no-console
  console.log(cron.metadata.id);

  await hatchet.crons.delete(cron);
}

if (require.main === module) {
  main();
}
