import { hatchet } from '../../hatchet-client';

// > Step 02 Schedule One Time
// Schedule a one-time run at a specific time.
const runAt = new Date(Date.now() + 60 * 60 * 1000);
await hatchet.scheduled.create('ScheduledWorkflow', { triggerAt: runAt, input: {} });
// !!
