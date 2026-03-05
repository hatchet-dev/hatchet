/** Mock embedding - no external API dependencies */

export function embed(text: string): number[] {
  return Array(64).fill(0.1);
}
