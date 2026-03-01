let orchestratorCallCount = 0;

export interface ToolCallResponse {
  done: boolean;
  content: string;
  toolCall?: { name: string; args: { task: string } };
}

export function mockOrchestratorLlm(messages: Array<{ role: string; content: string }>): ToolCallResponse {
  orchestratorCallCount++;
  switch (orchestratorCallCount) {
    case 1:
      return { done: false, content: '', toolCall: { name: 'research', args: { task: 'Find key facts about the topic' } } };
    case 2:
      return { done: false, content: '', toolCall: { name: 'writing', args: { task: 'Write a summary from the research' } } };
    default:
      return { done: true, content: 'Here is the final report combining research and writing.' };
  }
}

export function mockSpecialistLlm(task: string, role: string): string {
  return `[${role}] Completed: ${task}`;
}
