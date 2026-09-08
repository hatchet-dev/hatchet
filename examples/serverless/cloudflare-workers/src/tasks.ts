/**
 * The tasks this endpoint serves. Names and actions are un-prefixed; the operator applies the
 * endpoint's namespace (`<uuid>_echo`, `<uuid>_echo:echo`) when it registers them.
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
 * Durable: memoizes a timestamp, sleeps 3 seconds and returns both. The sleep is longer than
 * the endpoint's inline wait budget, so the first invocation evicts itself and the engine
 * re-invokes the task when the sleep is over; the second invocation gets the memoized
 * timestamp from the event log and finishes.
 */
export const sleepThenEcho = hatchet.durableTask({
  name: "sleep-then-echo",
  executionTimeout: "5m",
  fn: async (input: { message: string }, ctx) => {
    const startedAt = await ctx.now();

    await ctx.sleepFor("3s");

    return {
      echo: input.message,
      startedAt: startedAt.toISOString(),
      finishedAt: new Date().toISOString(),
      invocation: ctx.invocationCount,
    };
  },
});

export const workflows = [echo, sleepThenEcho];
