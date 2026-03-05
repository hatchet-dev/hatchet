export function mockGenerateContent(message: string): string {
  return `Here is a helpful response to: ${message}`;
}

export function mockSafetyCheck(message: string): { safe: boolean; reason: string } {
  if (message.toLowerCase().includes('unsafe')) {
    return { safe: false, reason: 'Content flagged as potentially unsafe.' };
  }
  return { safe: true, reason: 'Content is appropriate.' };
}

export function mockEvaluate(content: string): { score: number; approved: boolean } {
  const score = content.length > 20 ? 0.85 : 0.3;
  return { score, approved: score >= 0.7 };
}
