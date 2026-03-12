/** Mock LLM - no external API dependencies */

export function generate(prompt: string): { content: string; valid: boolean } {
  return { content: `Generated for: ${prompt.slice(0, 50)}...`, valid: true };
}
