/**
 * The tasks this endpoint serves. The operator registers them under these names, the way a
 * worker would (`echo`, action `echo:echo`), so they are triggered by the names declared here.
 * They are the same tasks as examples/serverless/cloudflare-workers/src/tasks.ts, so one
 * endpoint registration works for either runtime.
 */
import { hatchet } from "@hatchet-dev/serverless";

/** Non-durable: returns its input with the run id and retry count from the context. */
export const echo = hatchet.task({
  name: "echo",
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

export const workflows = [echo, sleepThenEcho];
