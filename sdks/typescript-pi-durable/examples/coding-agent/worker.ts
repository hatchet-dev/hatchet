// The worker that hosts the example's sessions. Killing this process is the crash test: Hatchet retries the session's
// run on the next worker, and pi-durable resumes it from its storage.
import { codingSession, hatchet } from './agent.js';

const worker = await hatchet.worker('pi-session-worker', { workflows: [codingSession], slots: 20 });
await worker.start();
