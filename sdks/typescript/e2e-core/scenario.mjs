// The scenario run.mjs executes from a scratch package. It must reach the engine over
// HTTP/1.1 only, since that is what fetch gets from a serverless runtime and the Connect
// path this client depends on has to work without HTTP/2; an undici Agent with HTTP/2
// disabled is passed in so the constraint holds whatever the runtime's fetch negotiates by
// default. The event is pushed only after the direct run completed, so that the Go test can
// assert the worker saw the two runs in that order.
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
