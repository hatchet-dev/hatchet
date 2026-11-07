import { hatchet } from '../hatchet-client';

// > Run a command in a sandbox
export const session = hatchet.durableTask({
  name: 'agent-session',
  executionTimeout: '1h',
  fn: async (input: { conversationId: string }) => {
    const sandbox = await hatchet.sandbox.create({
      id: input.conversationId,
      provider: 'e2b',
      metadata: { conversation: input.conversationId },
    });

    await sandbox.files.write('main.py', 'print("hello from the sandbox")');
    const result = await sandbox.exec('python3 main.py');

    await sandbox.kill();
    return { stdout: result.stdout, exitCode: result.exitCode };
  },
});
