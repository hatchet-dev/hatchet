import { Snippet, LANGUAGE_MAP } from '../../types';
import { ContentProcessor } from '../processor.interface';

const getFileName = (name: string) => {
  const extension = name.split('.').pop();
  const fileName = name.split('.').slice(0, -1).join('-');
  return { extension, fileName };
};

/**
 * Processes content by creating a TypeScript string
 * that exports a default Snippet with that content.
 */
const processSnippet: ContentProcessor = async ({ path, name, content }) => {
  const { extension, fileName } = getFileName(name);

  const language =
    extension && extension in LANGUAGE_MAP
      ? LANGUAGE_MAP[extension as keyof typeof LANGUAGE_MAP]
      : 'unknown';

  // Create a Snippet object
  const snippet: Snippet = {
    language,
    content,
    source: path,
  };

  // Generate TypeScript content that exports the snippet
  const tsContent = `import { Snippet } from '@/types';

const snippet: Snippet = ${JSON.stringify(snippet)};

export default snippet;
`;

  return {
    filename: `${fileName}.ts`,
    content: tsContent,
  };
};

export const snippetProcessor = {
  process: processSnippet,
  outDir: 'snips',
};
