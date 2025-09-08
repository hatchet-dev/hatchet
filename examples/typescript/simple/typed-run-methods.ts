import { simple } from './workflow';

async function main() {
  // > Run methods
  const input = { Message: 'Hello, World!' };

  // run now
  const result = await simple.run(input);
  const runReference = await simple.runNoWait(input);

  // or in the future
  const runAt = new Date(new Date().setHours(12, 0, 0, 0) + 24 * 60 * 60 * 1000);
  const scheduled = await simple.schedule(runAt, input);
  const cron = await simple.cron('simple-daily', '0 0 * * *', input);
}

async function runFlavors() {
  // > Run method flavors
  const input = { Message: 'Hello, World!' };

  // Run workflow and wait for the result
  const result = await simple.run(input);

  // Enqueue workflow to be executed asynchronously
  const runReference = await simple.runNoWait(input);
}
