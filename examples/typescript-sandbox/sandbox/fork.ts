import { hatchet } from '../hatchet-client';

// > Fork a sandbox
export async function fork(repo: string) {
  const base = await hatchet.sandbox.create({ provider: 'e2b', metadata: { repo } });
  await base.exec(`git clone ${repo} . && npm ci`);
  await base.kill();

  const [info] = await hatchet.sandbox.list({ sandboxId: base.sandboxId });
  const from = info.latestSnapshot!;

  const [a, b] = await Promise.all([
    hatchet.sandbox.create({ from, inheritParentMetadata: true }),
    hatchet.sandbox.create({ from, inheritParentMetadata: true }),
  ]);

  return [a.sandboxId, b.sandboxId];
}
