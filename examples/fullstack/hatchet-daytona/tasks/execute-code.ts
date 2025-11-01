import { hatchet, daytona } from '../clients';

export type ExecuteCodeInput = {
  code: string;
};

export const execute = hatchet.task({
  name: 'execute-code',
  retries: 3,
  fn: async (input: ExecuteCodeInput) => {

    // Create the Sandbox instance
    const sandbox = await daytona.create({
      user: 'hatchet',
      language: 'python',
    });
    
    try {
      // Run the provided code securely inside the Sandbox
      const response = await sandbox.process.codeRun(input.code);
      
      console.log('Code execution result:', response.result);
      
      return {
        result: response.result.trim()
      };
    } catch (error) {
      console.error('Code execution failed:', error);
      return {
        result: `Error executing code: ${error instanceof Error ? error.message : String(error)}`
      };
    } finally {
      // Stop the Sandbox
      await sandbox.stop();
    }
  },
});
