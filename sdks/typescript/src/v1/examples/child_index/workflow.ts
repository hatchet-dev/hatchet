import { hatchet } from '../hatchet-client';

export const childIndexChild = hatchet.task({
  name: 'child-index-child',
  fn: async (input: { tag: string }) => {
    return { tag: input.tag };
  },
});

export const childIndexParent = hatchet.workflow<{ n: number }>({
  name: 'child-index-parent',
});

childIndexParent.task({
  name: 'mixed-spawn',
  executionTimeout: '3m',
  fn: async (input: { n: number }, ctx) => {
    const { n } = input;

    const ctxResults: string[] = [];
    for (let i = 0; i < n; i++) {
      const result = await ctx.runChild(childIndexChild, { tag: `ctx-${i}` });
      ctxResults.push(result.tag);
    }

    const refs = await Promise.all(
      Array.from({ length: n }, (_, i) => childIndexChild.runNoWait({ tag: `rnw-${i}` }))
    );

    const runIds = await Promise.all(refs.map((r) => r.runId));
    const outputs = await Promise.all(refs.map((r) => r.output));
    const rnwResults = outputs.map((o) => o.tag);

    return {
      ctxResults,
      rnwResults,
      runIds,
    };
  },
});
