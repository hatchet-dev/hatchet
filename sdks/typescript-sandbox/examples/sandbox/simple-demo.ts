import { hatchet } from '../hatchet-client';

// > Simple demo
export const checkUpgrade = hatchet.durableTask({
  name: 'check-upgrade',
  executionTimeout: '1h',
  fn: async (input: { repo: string }) => {
    const sandbox = await hatchet.sandbox.create({
      provider: 'e2b',
      metadata: { repo: input.repo },
    });
    await sandbox.exec(`git clone ${input.repo} . && npm ci`);
    const before = await sandbox.exec('npm test');
    await sandbox.kill();

    const [info] = await hatchet.sandbox.list({ sandboxId: sandbox.sandboxId });
    const fork = await hatchet.sandbox.create({ from: info.latestSnapshot! });
    const after = await fork.exec('npm update && npm test');
    await fork.kill();

    return { passesToday: before.exitCode === 0, passesUpgraded: after.exitCode === 0 };
  },
});
// !!
