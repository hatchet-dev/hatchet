import sleep from '@hatchet-dev/typescript-sdk/util/sleep';
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

// Simulates recursive hierarchy of entities
// (fund → portfolio → project) where the orchestrator walks bottom-up,
// calling scenarioWorkflow.runNoWait() for each node and waiting for it
// to complete before processing the parent.

type TreeNode = {
  id: string;
  children: TreeNode[];
};

function buildTree(): TreeNode {
  // Root → Fund → Portfolio → Portfolio → Project structure
  // ~30 total nodes to keep the test manageable
  return {
    id: 'fund-root',
    children: [
      {
        id: 'fund-a',
        children: [
          {
            id: 'portfolio-a1',
            children: [
              { id: 'project-a1a', children: [] },
              { id: 'project-a1b', children: [] },
              { id: 'project-a1c', children: [] },
            ],
          },
          {
            id: 'portfolio-a2',
            children: [
              { id: 'project-a2a', children: [] },
              { id: 'project-a2b', children: [] },
            ],
          },
        ],
      },
      {
        id: 'fund-b',
        children: [
          {
            id: 'portfolio-b1',
            children: [
              { id: 'project-b1a', children: [] },
              { id: 'project-b1b', children: [] },
              { id: 'project-b1c', children: [] },
              { id: 'project-b1d', children: [] },
            ],
          },
          {
            id: 'portfolio-b2',
            children: [
              { id: 'project-b2a', children: [] },
              { id: 'project-b2b', children: [] },
            ],
          },
        ],
      },
      {
        id: 'fund-c',
        children: [
          {
            id: 'portfolio-c1',
            children: [
              { id: 'project-c1a', children: [] },
              { id: 'project-c1b', children: [] },
              { id: 'project-c1c', children: [] },
            ],
          },
        ],
      },
    ],
  };
}

function countNodes(node: TreeNode): number {
  return 1 + node.children.reduce((sum, c) => sum + countNodes(c), 0);
}

export const scenarioTask = hatchet.task({
  name: 'child-index-scenario',
  fn: async (input: { scenarioId: string }) => {
    await sleep(100);
    return { scenarioId: input.scenarioId, status: 'completed' };
  },
});

export const orchestratorTask = hatchet.task({
  name: 'child-index-orchestrator',
  executionTimeout: '5m',
  fn: async (input: {}, ctx) => {
    const tree = buildTree();
    const totalExpected = countNodes(tree);
    const collectedRunIds: string[] = [];
    const collectedScenarios: string[] = [];

    // Recursive bottom-up traversal — mirrors Evan's runChildScenarioFn pattern.
    // Process children first, wait for completion, then process this node.
    async function processNode(node: TreeNode): Promise<void> {
      for (const child of node.children) {
        await processNode(child);
      }

      const runRef = await scenarioTask.runNoWait({
        scenarioId: node.id,
      });
      const runId = await runRef.runId;
      collectedRunIds.push(runId);
      collectedScenarios.push(node.id);

      await runRef.output;
    }

    await processNode(tree);

    const uniqueRunIds = new Set(collectedRunIds);

    return {
      total: collectedRunIds.length,
      expected: totalExpected,
      uniqueRunIds: uniqueRunIds.size,
      hasDuplicates: uniqueRunIds.size !== collectedRunIds.length,
      scenarios: collectedScenarios,
    };
  },
});
