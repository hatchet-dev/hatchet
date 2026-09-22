// The core client scenario run.mjs executes from a scratch package: it reaches the engine
// through the fetch transport over HTTP/1.1 only (an undici Agent that never negotiates
// HTTP/2, which is what fetch does from a serverless runtime), triggers the echo workflow a
// Go worker serves, waits for the result by polling, reads the run back and pushes an event
// the same workflow is bound to.
import { Agent, fetch as undiciFetch } from 'undici';
import { Hatchet } from '@hatchet-dev/typescript-sdk/core';
import { declarations } from '@hatchet-dev/typescript-sdk/edge';

function required(env, name) {
  const value = env[name];
  if (!value) throw new Error(`${name} is required`);
  return value;
}

export async function run(env) {
  const token = required(env, 'HATCHET_CLIENT_TOKEN');
  const hostPort = required(env, 'HATCHET_CLIENT_HOST_PORT');
  const workflowName = env.HATCHET_E2E_WORKFLOW ?? 'tscore-echo';
  const eventKey = env.HATCHET_E2E_EVENT ?? 'tscore:echo';

  const dispatcher = new Agent({ allowH2: false });
  const fetch = (input, init) => undiciFetch(input, { ...init, dispatcher });

  const hatchet = new Hatchet({
    token,
    hostPort,
    tls: { strategy: 'none' },
    fetch,
    logLevel: 'WARN',
  });

  const { task } = declarations();
  const echo = task({ name: workflowName, fn: (input) => input });

  const message = `hello from core ${Date.now()}`;
  const ref = await hatchet.runNoWait(echo, { message });
  const output = await ref.result({ timeoutMs: 60_000 });
  const details = await hatchet.runs.get(ref);
  const event = await hatchet.events.push(eventKey, { message: `event ${message}` });

  await dispatcher.close();

  return {
    runId: ref.workflowRunId,
    output,
    status: details.status,
    done: details.done,
    eventId: event.eventId,
    eventKey: event.key,
  };
}
