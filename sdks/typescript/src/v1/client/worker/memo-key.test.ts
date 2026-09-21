import { createHash } from 'crypto';
import { computeMemoKey } from './context';

// The Node implementation this replaced; recorded event logs carry keys produced by it.
function legacyComputeMemoKey(taskRunExternalId: string, args: readonly unknown[]): Uint8Array {
  const h = createHash('sha256');
  h.update(taskRunExternalId);
  h.update(JSON.stringify(args));
  return new Uint8Array(h.digest());
}

describe('computeMemoKey', () => {
  const cases: Array<[string, readonly unknown[]]> = [
    ['task-1', ['now']],
    ['5d3c8e5a-6f1a-4d0e-9d7b-2d6c4a1b0e9f', ['now']],
    ['task-2', ['memo', { a: 1, b: [1, 2, 3] }, 'unicode: héllo ✓']],
    ['', []],
  ];

  it.each(cases)('matches the createHash implementation for %s', async (id, args) => {
    const key = await computeMemoKey(id, args);
    expect(key).toBeInstanceOf(Uint8Array);
    expect(key).toHaveLength(32);
    expect(Buffer.from(key).toString('hex')).toBe(
      Buffer.from(legacyComputeMemoKey(id, args)).toString('hex')
    );
  });

  it('is stable for a known input', async () => {
    const key = await computeMemoKey('task-1', ['now']);
    expect(Buffer.from(key).toString('hex')).toBe(
      createHash('sha256').update('task-1').update('["now"]').digest('hex')
    );
  });
});
