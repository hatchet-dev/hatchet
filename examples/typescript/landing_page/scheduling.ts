import { simple } from './flow-control';

// ❓ Schedules and Crons
const tomorrow = new Date(Date.now() + 1000 * 60 * 60 * 24);
const scheduled = simple.schedule(tomorrow, {
  Message: 'Hello, World!',
});

const cron = simple.cron('every-day', '0 0 * * *', {
  Message: 'Hello, World!',
});
