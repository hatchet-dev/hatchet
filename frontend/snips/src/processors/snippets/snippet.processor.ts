import { Snippet } from '../../types';
import { ContentProcessor } from '../processor.interface';

/**
 * Processes content by creating a TypeScript string
 * that exports a default Snippet with that content.
 */
export const snippetProcessor: ContentProcessor = (content: string): string => {
  // Create a Snippet object
  const snippet: Snippet = {
    content,
  };

  // Generate TypeScript content that exports the snippet
  const tsContent = `
import { Snippet } from '@/types';

const snippet: Snippet = {
  content: \`${content.replace(/`/g, '\\`')}\`
};

export default snippet;
`;

  return tsContent;
};
