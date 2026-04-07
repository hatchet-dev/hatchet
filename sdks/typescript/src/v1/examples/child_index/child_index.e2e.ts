import { makeE2EClient } from '../__e2e__/harness';
import { childIndexParent } from './workflow';

describe('child-index-e2e', () => {
  const hatchet = makeE2EClient();

  it('produces unique children when mixing ctx.runChild and workflow.runNoWait', async () => {
    const N = 5;
    const result = await childIndexParent.run({ n: N });
    const data = (result as any)['mixed-spawn'];

    const { ctxResults } = data;
    const { rnwResults } = data;
    const { runIds } = data;

    expect(ctxResults).toHaveLength(N);
    expect(rnwResults).toHaveLength(N);
    expect(runIds).toHaveLength(N);

    for (let i = 0; i < N; i++) {
      expect(ctxResults[i]).toBe(`ctx-${i}`);
      expect(rnwResults[i]).toBe(`rnw-${i}`);
    }

    const uniqueRunIds = new Set(runIds);
    expect(uniqueRunIds.size).toBe(N);
  }, 120_000);

  it('produces unique children when using only runNoWait in parallel', async () => {
    const N = 8;
    const result = await childIndexParent.run({ n: N });
    const data = (result as any)['mixed-spawn'];

    const allTags = [...data.ctxResults, ...data.rnwResults];
    const uniqueTags = new Set(allTags);

    expect(uniqueTags.size).toBe(N * 2);

    const uniqueRunIds = new Set(data.runIds as string[]);
    expect(uniqueRunIds.size).toBe(N);
  }, 120_000);
});
