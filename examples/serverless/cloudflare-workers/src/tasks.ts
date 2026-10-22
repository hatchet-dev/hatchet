/**
 * The tasks this endpoint serves. The operator registers them under these names, the way a
 * worker would (`echo`, action `echo:echo`), so they are triggered by the names declared here.
 */
import { clientFromContext, hatchet } from "@hatchet-dev/serverless";

/**
 * Non-durable and pure: a function of its input (plus the run id and retry count the request
 * carries) that never talks to Hatchet, so it is declared with `hatchetClient: false` and the
 * operator delivers it over a signed POST rather than the invocation websocket.
 */
export const echo = hatchet.task({
  name: "echo",
  hatchetClient: false,
  fn: async (input: { message: string }, ctx) => ({
    echo: input.message,
    workflowRunId: ctx.workflowRunId(),
    retryCount: ctx.retryCount(),
  }),
});

/**
 * Durable: memoizes a timestamp, sleeps 3 seconds, then spawns `echo` as a child run and waits
 * for its output. Both the sleep and the child run go over the relay as durable events, and
 * each is longer than the endpoint's inline wait budget, so the invocation evicts itself while
 * waiting and the engine re-invokes the task when the awaited entry is satisfied. Every
 * re-invocation replays the memoized timestamp, the finished sleep and the child's output
 * from the event log, so the function runs from the top without repeating any of them.
 */
export const sleepThenEcho = hatchet.durableTask({
  name: "sleep-then-echo",
  executionTimeout: "5m",
  fn: async (input: { message: string }, ctx) => {
    const startedAt = await ctx.now();

    await ctx.sleepFor("3s");

    const child = await ctx.spawnChild(echo, { message: `${input.message} (from child)` });

    return {
      echo: input.message,
      child,
      startedAt: startedAt.toISOString(),
      finishedAt: new Date().toISOString(),
      invocation: ctx.invocationCount,
    };
  },
});

/**
 * Non-durable, with a child: triggers `echo` through the Worker's Hatchet client (a unary call
 * with the token from HATCHET_CLIENT_TOKEN) and awaits its result on the invocation websocket,
 * where the operator relays the run's terminal event from the engine. The socket is the
 * default delivery for a task that talks to Hatchet, so nothing is declared; over a plain POST
 * the wait would throw. Unlike the durable task it never evicts: it waits inline, as
 * invocation 1.
 */
export const parentEcho = hatchet.task({
  name: "parent-echo",
  executionTimeout: "5m",
  fn: async (input: { message: string }, ctx) => {
    const child = await ctx.runChild(echo, { message: `${input.message} (from parent)` });

    return {
      echo: input.message,
      child,
      workflowRunId: ctx.workflowRunId(),
    };
  },
});

/**
 * Non-durable, on the socket by default since it talks to Hatchet: publishes a handful of
 * progress messages to a durable stream topic through the Worker's Hatchet client
 * (`clientFromContext(ctx)`, the same core client `ctx.runChild` triggers through) and
 * returns the topic. The topic is the input's `topic` or
 * one derived from the run id, so a consumer can follow a run it triggered:
 * `scripts/consume-stream.ts` reads it with the Node SDK's `hatchet.streams.events(topic)`.
 * Without HATCHET_CLIENT_TOKEN the client call throws and the task fails without retry.
 */
export const streamProgress = hatchet.task({
  name: "stream-progress",
  fn: async (input: { topic?: string; steps?: number }, ctx) => {
    const client = clientFromContext(ctx);
    const topic = input.topic ?? `progress-${ctx.workflowRunId()}`;
    const steps = Math.min(Math.max(input.steps ?? 5, 1), 20);

    for (let step = 1; step <= steps; step += 1) {
      await client.streams.publish(
        topic,
        JSON.stringify({ step, of: steps, message: `step ${step} of ${steps}` }),
      );
    }

    await client.streams.publish(topic, JSON.stringify({ done: true }));

    return { topic, published: steps + 1 };
  },
});

/**
 * Non-durable, on the socket by default: publishes `count` messages and a final
 * `{"done": true}` to a durable stream topic through the Worker's Hatchet client, then reads the
 * topic back in the same invocation with `client.streams.events(topic)` and returns what it
 * read. Reading a topic is a server stream, which the core client does not carry on its own;
 * inside a task the package serves it over the invocation websocket: the operator opens the
 * engine's `/v1.V1Streams/Subscribe` stream and relays its messages back, the way a child's
 * result is awaited. The read starts at the oldest retained message and stops at the done
 * message, which closes the stream and frees its slot on the socket. It needs an operator
 * that relays the subscription (an older one closes it with code 12) and a tenant with the
 * `durable_streams` entitlement (code 7); either failure is a `StreamClosedError`.
 */
export const streamEcho = hatchet.task({
  name: "stream-echo",
  fn: async (input: { topic?: string; message?: string; count?: number }, ctx) => {
    const client = clientFromContext(ctx);
    const topic = input.topic ?? `echo-${ctx.workflowRunId()}`;
    const count = Math.min(Math.max(input.count ?? 3, 1), 20);
    const message = input.message ?? "hello";

    for (let index = 1; index <= count; index += 1) {
      await client.streams.publish(topic, JSON.stringify({ index, message }));
    }

    await client.streams.publish(topic, JSON.stringify({ done: true }));

    const decoder = new TextDecoder();
    const read: string[] = [];

    for await (const event of client.streams.events(topic)) {
      const text = decoder.decode(event.payload);
      read.push(text);

      if (JSON.parse(text).done === true) {
        break;
      }
    }

    return { topic, published: count + 1, read };
  },
});

export const workflows = [echo, sleepThenEcho, parentEcho, streamProgress, streamEcho];
