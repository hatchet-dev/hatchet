import { hatchet } from '../hatchet-client';

// > Fork a sandbox
export async function fork(repo: string) {
  const base = await hatchet.sandbox.create({ provider: 'e2b', metadata: { repo } });
  await base.exec(`git clone ${repo} . && npm ci`);
  await base.kill();

  const [a, b] = await Promise.all([
    hatchet.sandbox.create({ from: base.sandboxId, inheritParentMetadata: true }),
    hatchet.sandbox.create({ from: base.sandboxId, inheritParentMetadata: true }),
  ]);

  return [a.sandboxId, b.sandboxId];
}
