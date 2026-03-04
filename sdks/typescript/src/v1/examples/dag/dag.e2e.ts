import { makeE2EClient } from '../__e2e__/harness';
import { dag } from './workflow';

describe('dag-e2e', () => {
  const hatchet = makeE2EClient();

  it('runs the DAG and produces expected output', async () => {
    const res = await dag.run({
      Message: 'hello',
    });

    // Ensure parent output access and transform happened
    expect(res.reverse.Original).toBe('hello');
    expect(res.reverse.Transformed).toBe('olleh');
  }, 60_000);
});
