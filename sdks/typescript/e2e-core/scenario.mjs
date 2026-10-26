// The scenario run.mjs executes from a scratch package. It must reach the engine over
// HTTP/1.1 only, since that is what fetch gets from a serverless runtime and the Connect
// path this client depends on has to work without HTTP/2; an undici Agent with HTTP/2
// disabled is passed in so the constraint holds whatever the runtime's fetch negotiates by
// default. The event is pushed only after the direct run completed, so that the Go test can
// assert the worker saw the two runs in that order. The durable stream step publishes two
// messages to a topic named after the run and reports the topic's metadata, so the Go test
// can check both were stored. The REST step lists the workflow's runs through the API server,
// whose URL comes from the environment since the harness token names a server the test does
// not run, and reports whether the direct run was among them.
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
  const apiUrl = required(env, 'HATCHET_CLIENT_API_URL');
  const workflowName = env.HATCHET_E2E_WORKFLOW ?? 'tscore-echo';
  const eventKey = env.HATCHET_E2E_EVENT ?? 'tscore:echo';

  const dispatcher = new Agent({ allowH2: false });
  const fetch = (input, init) => undiciFetch(input, { ...init, dispatcher });

  const hatchet = new Hatchet({
    token,
    hostPort,
    apiUrl,
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

  const streamTopic = `tscore-stream-${ref.workflowRunId}`;
  await hatchet.streams.publish(streamTopic, 'first');
  await hatchet.streams.publish(streamTopic, new TextEncoder().encode('second'));
  const topic = await hatchet.streams.topicMetadata(streamTopic);
  const listed = await findListedRun(hatchet, workflowName, ref.workflowRunId);

  await dispatcher.close();

  return {
    runId: ref.workflowRunId,
    output,
    status: details.status,
    done: details.done,
    eventId: event.eventId,
    eventKey: event.key,
    streamTopic,
    streamMessageCount: topic.messageCount,
    streamLatestCursor: topic.latestCursor ?? '',
    listedRun: listed !== undefined,
    listedRunStatus: listed?.status ?? '',
  };
}

// The run list is served from the OLAP tables, which are written a moment after the run
// completes, so the listing is retried until the run shows up.
async function findListedRun(hatchet, workflowName, runId, attempts = 60) {
  for (let attempt = 0; attempt < attempts; attempt += 1) {
    const { rows } = await hatchet.runs.list({
      workflowNames: [workflowName],
      since: new Date(Date.now() - 10 * 60_000),
      onlyTasks: false,
      limit: 100,
    });
    const row = rows?.find(
      (r) =>
        r.metadata.id === runId || r.workflowRunExternalId === runId || r.taskExternalId === runId
    );
    if (row) return row;
    await new Promise((resolve) => setTimeout(resolve, 500));
  }
  return undefined;
}
