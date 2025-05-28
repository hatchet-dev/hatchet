import { hatchet } from '../hatchet-client';
import { simple } from './workflow';

async function main() {
  // > Create
  const cron = await simple.cron('simple-daily', '0 0 * * *', {
    Message: 'hello',
  });

  // 👀 Get the cron ID of the workflow
  // it may be helpful to store the cron ID of the workflow
  // in a database or other persistent storage for later use
  const cronId = cron.metadata.id;
  console.log(cronId);
  // !!

  // // > Delete a Cron Trigger
  // await hatchet.crons.delete(cron);
  // // !!

  // > List Cron Triggers
  // const crons = await hatchet.crons.list({
  //   workflow: simple,
  // });
  // console.log(crons);
  // !!
}

if (require.main === module) {
  main();
}
