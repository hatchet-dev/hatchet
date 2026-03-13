// Third-party integration - requires: pnpm add @google-cloud/vision
// See: /guides/document-processing

import { ImageAnnotatorClient } from '@google-cloud/vision';

const client = new ImageAnnotatorClient();

// > Google Vision usage
export async function parseDocument(content: Buffer): Promise<string> {
  const [result] = await client.documentTextDetection({ image: { content } });
  return result.fullTextAnnotation?.text ?? '';
}
