import { streamTask } from './workflow';

// > Step 03 Subscribe Client
// Client triggers the task and subscribes to the stream.
async function runAndSubscribe() {
  const run = await streamTask.run({});
  for await (const chunk of run.stream()) {
    console.log(chunk);
  }
}
