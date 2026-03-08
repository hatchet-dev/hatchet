export function mockClassify(message: string): string {
  const lower = message.toLowerCase();
  if (lower.includes('bug') || lower.includes('error') || lower.includes('help')) return 'support';
  if (lower.includes('price') || lower.includes('buy') || lower.includes('plan')) return 'sales';
  return 'other';
}

export function mockReply(message: string, role: string): string {
  switch (role) {
    case 'support':
      return `[Support] I can help with that technical issue. Let me look into: ${message}`;
    case 'sales':
      return `[Sales] Great question about pricing! Here's what I can tell you about: ${message}`;
    default:
      return `[General] Thanks for reaching out. Regarding: ${message}`;
  }
}
