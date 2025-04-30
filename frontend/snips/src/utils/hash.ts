export const hash = async (content: string): Promise<string> => {
  // Use the Web Crypto API to generate a SHA-256 digest
  const msgUint8 = new TextEncoder().encode(content);
  const hashBuffer = await crypto.subtle.digest('SHA-256', msgUint8);

  // Convert the ArrayBuffer to hex string
  return Array.from(new Uint8Array(hashBuffer))
    .map((b) => b.toString(16).padStart(2, '0'))
    .slice(0, 8)
    .join('');
};
