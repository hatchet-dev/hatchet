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
 * Durable: memoizes a timestamp, sleeps 3 seconds and returns both. The healthcheck advertises
 * it, but this version of @hatchet-dev/serverless cannot serve durable tasks (it reports
 * `durable.supported: false`, so the operator never assigns it). The durable relay is the next
 * phase; the declaration stays so the registration is already the final one.
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
    };
  },
});

export const workflows = [echo, sleepThenEcho];
