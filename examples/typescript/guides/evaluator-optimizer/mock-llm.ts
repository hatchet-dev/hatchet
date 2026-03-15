let generateCount = 0;

export function mockGenerate(prompt: string): string {
  generateCount++;
  if (generateCount === 1) {
    return 'Check out our product! Buy now!';
  }
  return 'Discover how our tool saves teams 10 hours/week. Try it free.';
}

export function mockEvaluate(draft: string): { score: number; feedback: string } {
  if (draft.length < 40) {
    return { score: 0.4, feedback: 'Too short and pushy. Add a specific benefit and soften the CTA.' };
  }
  return { score: 0.9, feedback: 'Clear value prop, appropriate tone.' };
}
