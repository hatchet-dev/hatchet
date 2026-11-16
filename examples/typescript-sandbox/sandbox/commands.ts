import { hatchet } from '../hatchet-client';

// > Run a command
export async function runTests(sandboxId: string) {
  const sandbox = await hatchet.sandbox.connect(sandboxId);

  const result = await sandbox.exec('npm test', { timeoutMs: 5 * 60 * 1000 });

  if (result.exitCode === 124) {
    throw new Error(`tests timed out:\n${result.stderr}`);
  }

  return { passed: result.exitCode === 0, output: result.stdout };
}
