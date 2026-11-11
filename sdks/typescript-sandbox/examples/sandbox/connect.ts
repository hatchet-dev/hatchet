import { hatchet } from '../hatchet-client';

// > Connect to a sandbox
export async function gitStatus(sandboxId: string) {
  const sandbox = await hatchet.sandbox.connect(sandboxId);
  const result = await sandbox.exec('git status --short');
  return result.stdout;
}
// !!
