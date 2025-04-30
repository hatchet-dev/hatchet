import { Priority } from '@hatchet/v1';
import { priority } from './workflow';

/* eslint-disable no-console */
async function main() {
  try {
    console.log('running priority workflow');

    // > Run a Task with a Priority
    const run = priority.run(new Date(Date.now() + 60 * 60 * 1000), { priority: Priority.HIGH });
    // !!

    // > Schedule and cron
    const scheduled = priority.schedule(
      new Date(Date.now() + 60 * 60 * 1000),
      {},
      { priority: Priority.HIGH }
    );
    const delayed = priority.delay(60 * 60 * 1000, {}, { priority: Priority.HIGH });
    const cron = priority.cron(
      `daily-cron-${Math.random()}`,
      '0 0 * * *',
      {},
      { priority: Priority.HIGH }
    );
    // !!

    const [scheduledResult, delayedResult] = await Promise.all([scheduled, delayed]);
    console.log('scheduledResult', scheduledResult);
    console.log('delayedResult', delayedResult);
    // !!
  } catch (e) {
    console.log('error', e);
  }
}

if (require.main === module) {
  main()
    .catch(console.error)
    .finally(() => process.exit(0));
}
