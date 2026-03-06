// Third-party integration - requires: pnpm add tesseract.js
// See: /guides/document-processing

import Tesseract from 'tesseract.js';

// > Tesseract usage
export async function parseDocument(content: Buffer): Promise<string> {
  const { data } = await Tesseract.recognize(content);
  return data.text;
}
