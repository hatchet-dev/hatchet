import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
import { EvictionPolicy } from '@hatchet-dev/typescript-sdk/v1';
import { hatchet } from '../hatchet-client';

const WORKFLOW_PREFIX = 'durable-callback-ordering';

const EVICTION_POLICY: EvictionPolicy = {
  ttl: 250,
  allowCapacityEviction: true,
  priority: 0,
};

export type LeafInput = {
  mid: number;
  branch: number;
  delayMs: number;
  generation: number;
};

export type LeafOutput = {
  mid: number;
  branch: number;
  generation: number;
};

export type MidInput = {
  mid: number;
  branches: number;
  childDelayMs: number;
  delayStepMs: number;
};

export type MidOutput = {
  mid: number;
  completedBranches: number[];
  invocationCount: number;
};

export type RootInput = {
  durables?: number;
  branches?: number;
  childDelayMs?: number;
  delayStepMs?: number;
};

export type RootOutput = {
  rootInvocationCount: number;
  midInvocationCounts: number[];
  completedMids: number[];
};

export const ROOT_DEFAULTS = {
  durables: 4,
  branches: 8,
  childDelayMs: 1_500,
  delayStepMs: 3,
};

export const callbackOrderingLeaf = hatchet.task<LeafInput, LeafOutput>({
  name: `${WORKFLOW_PREFIX}-leaf`,
  executionTimeout: '1m',
  fn: async (input) => {
    await sleep(input.delayMs);
    return { mid: input.mid, branch: input.branch, generation: input.generation };
  },
});

export const callbackOrderingMid = hatchet.durableTask<MidInput, MidOutput>({
  name: `${WORKFLOW_PREFIX}-mid`,
  executionTimeout: '5m',
  retries: 0,
  evictionPolicy: EVICTION_POLICY,
  fn: async (input, ctx) => {
    // Staggered first-generation children complete out of spawn order, so each
    // branch's second-generation spawn is emitted in completion order. Replays
    // after eviction must re-deliver those completions in the recorded order or
    // the re-emitted spawn sequence diverges from the event log.
    const branch = async (branchIndex: number): Promise<number> => {
      const firstDelayMs =
        input.childDelayMs + (input.branches - branchIndex - 1) * input.delayStepMs;
      const first = await callbackOrderingLeaf.run({
        mid: input.mid,
        branch: branchIndex,
        delayMs: firstDelayMs,
        generation: 1,
      });
      const second = await callbackOrderingLeaf.run({
        mid: input.mid,
        branch: first.branch,
        delayMs: input.childDelayMs,
        generation: 2,
      });
      return second.branch;
    };

    const completed = await Promise.all(
      Array.from({ length: input.branches }, (_, branchIndex) => branch(branchIndex))
    );
    return {
      mid: input.mid,
      completedBranches: completed,
      invocationCount: ctx.invocationCount,
    };
  },
});

export const callbackOrderingRoot = hatchet.durableTask<RootInput, RootOutput>({
  name: `${WORKFLOW_PREFIX}-root`,
  executionTimeout: '10m',
  retries: 0,
  evictionPolicy: EVICTION_POLICY,
  fn: async (input, ctx) => {
    const durables = input.durables ?? ROOT_DEFAULTS.durables;
    const results = await Promise.all(
      Array.from({ length: durables }, (_, midIndex) =>
        callbackOrderingMid.run({
          mid: midIndex,
          branches: input.branches ?? ROOT_DEFAULTS.branches,
          childDelayMs: input.childDelayMs ?? ROOT_DEFAULTS.childDelayMs,
          delayStepMs: input.delayStepMs ?? ROOT_DEFAULTS.delayStepMs,
        })
      )
    );
    return {
      rootInvocationCount: ctx.invocationCount,
      midInvocationCounts: results.map((item) => item.invocationCount),
      completedMids: results.map((item) => item.mid),
    };
  },
});
