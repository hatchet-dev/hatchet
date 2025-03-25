import Hatchet from '../../sdk';
import { simpleWorkflow } from '../simple-worker';

const hatchet = Hatchet.init();

// This example assumes you have a worker already running
// and registered simpleWorkflow to it

async function main() {
  // ? Create
  // You can create dynamic scheduled runs programmatically via the API
  const createdScheduledRun = await hatchet.schedules.create(
    simpleWorkflow, // workflow object or string workflow id
    {
      triggerAt: new Date(Date.now() + 1000 * 60 * 60 * 24), // 24 hours from now
      input: {
        name: 'John Doe',
      },
      additionalMetadata: {
        customerId: '123',
      },
    }
  );
  const { id } = createdScheduledRun.metadata; // id which you can later use to reference the scheduled run
  // !!

  // ? Get
  // You can get a specific scheduled run by passing in the scheduled run id
  const scheduledRun = await hatchet.schedules.get(id);
  // !!

  // ? Delete
  // You can delete a scheduled run by passing the scheduled run object
  // or a scheduled run Id to the delete method
  await hatchet.schedules.delete(scheduledRun);
  // !!

  // ? List
  // You can list all scheduled runs by passing in a query object
  const scheduledRunList = await hatchet.schedules.list({
    offset: 0,
    limit: 10,
  });
  // !!
}

main();
