import fs from 'fs/promises';
import { getConfig } from '../../utils/config';
import { Snippet, LANGUAGE_MAP, Block, Highlight } from '../../types';
import { ContentProcessor, DirectoryProcessor, Processor } from '../processor.interface';
import path from 'path';

const TOKENS = {
  BLOCK: {
    START: '>',
    END: '!!',
  },
  HIGHLIGHT: {
    START: 'HH-',
  },
};

const getFileName = (name: string) => {
  const extension = name.split('.').pop();
  const fileName = name.split('.').slice(0, -1).join('-');
  return { extension, fileName };
};

const sanitizeContent = (content: string) => {
  const { REMOVAL_PATTERNS, REPLACEMENTS } = getConfig();

  let cleanedContent = content;

  REMOVAL_PATTERNS.forEach((pattern) => {
    cleanedContent = cleanedContent.replace(pattern.regex, '');
  });

  REPLACEMENTS.forEach((replacement) => {
    cleanedContent = cleanedContent.replace(replacement.from, replacement.to);
  });

  return cleanedContent;
};

const getCommentStyle = (language: string) => (language === 'python' ? '#' : '//');

const removeLine = (content: string): boolean => {
  const { REMOVAL_PATTERNS } = getConfig();
  return REMOVAL_PATTERNS.some((pattern) => content.match(pattern.regex));
};

const processBlocks = (content: string, language: string): { blocks: { [key: string]: Block } } => {
  const lines = content.split('\n');
  const blocks: { [key: string]: Block } = {};
  let currentBlock: { start: number; key: string } | null = null;
  let removedLines = 0;

  const commentStyle = getCommentStyle(language);

  lines.forEach((line, index) => {
    const trimmedLine = line.trim();
    const currentLineNumber = index + 1 - removedLines; // Adjust for removed lines

    if (trimmedLine.startsWith(`${commentStyle} ${TOKENS.BLOCK.START}`)) {
      const key = trimmedLine.replace(`${commentStyle} ${TOKENS.BLOCK.START}`, '').trim();
      currentBlock = { start: currentLineNumber + 1, key }; // Start on next line
    } else if (trimmedLine.startsWith(`${commentStyle} ${TOKENS.BLOCK.END}`) && currentBlock) {
      blocks[normalizeKey(currentBlock.key)] = {
        start: currentBlock.start,
        stop: currentLineNumber - 1, // -1 because we want the line before the !!
      };
      currentBlock = null;
    }

    if (removeLine(trimmedLine)) {
      removedLines++;
    }
  });

  return { blocks };
};

const normalizeKey = (key: string) =>
  key
    .toLowerCase()
    .replace(/ /g, '_')
    .replace(/[^a-z0-9_]/g, '');

const processHighlights = (content: string, language: string): { [key: string]: Highlight } => {
  const lines = content.split('\n');
  const highlights: { [key: string]: Highlight } = {};
  let removedLines = 0;

  const commentStyle = getCommentStyle(language);

  lines.forEach((line, index) => {
    const trimmedLine = line.trim();
    const currentLineNumber = index - removedLines;

    const highlightMatch = trimmedLine.match(
      new RegExp(`${commentStyle} ${TOKENS.HIGHLIGHT.START}([^ ]+) (\\d+)(?: (.*))?`),
    );
    if (highlightMatch) {
      const [, key, lineCountStr, stringsStr] = highlightMatch;
      const lineCount = parseInt(lineCountStr, 10);
      const strings = stringsStr ? stringsStr.split(',').map((s) => s.trim()) : [];

      // Calculate all line numbers to highlight
      const startLine = currentLineNumber + 1;
      const lines = Array.from({ length: lineCount }, (_, i) => startLine + i);

      highlights[normalizeKey(key)] = {
        lines,
        strings,
      };
    }

    if (removeLine(trimmedLine)) {
      removedLines++;
    }
  });

  return highlights;
};

const processBlocksAndHighlights = (
  content: string,
  language: string,
): { blocks: { [key: string]: Block }; highlights: { [key: string]: Highlight } } => {
  const { blocks } = processBlocks(content, language);
  const highlights = processHighlights(content, language);
  return { blocks, highlights };
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

  const cleanedContent = sanitizeContent(content);
  const { blocks, highlights } = processBlocksAndHighlights(content, language);

  // Create a Snippet object
  const snippet: Snippet = {
    language,
    content: cleanedContent,
    source: path,
    blocks,
    highlights,
  };

  // Generate TypeScript content that exports the snippet
  const tsContent = `import { Snippet } from '@/types';

const snippet: Snippet = ${JSON.stringify(snippet, null, 2)
    .replace(/'/g, "\\'") // First escape any single quotes
    .replace(/"/g, "'")};  // Then replace double quotes with single quotes

export default snippet;
`;

  return [
    {
      content: cleanedContent,
      outDir: 'examples',
    },
    {
      filename: `${fileName}.ts`,
      content: tsContent,
      outDir: 'snips',
    },
  ];
};

const processDirectory: DirectoryProcessor = async ({ dir }) => {
  if (!dir.includes('snips')) {
    return;
  }

  const entries = await fs.readdir(dir, { withFileTypes: true });
  const snippets = entries.filter(
    (entry) => entry.isFile() && entry.name.endsWith('.ts') && entry.name !== 'index.ts',
  );
  const directories = entries.filter((entry) => entry.isDirectory());

  if (snippets.length === 0 && directories.length === 0) {
    return;
  }

  // Generate import and export statements for files
  const fileImports = snippets.map((file) => {
    const baseName = file.name.replace('.ts', '').replace(/-/g, '_');
    return `import ${baseName} from './${file.name.replace('.ts', '')}';`;
  });

  const fileExports = snippets.map((file) => {
    const baseName = file.name.replace('.ts', '').replace(/-/g, '_');
    return `export { ${baseName} as ${baseName} }`;
  });

  // Generate import and export statements for directories
  const dirImports = directories.map((dir) => {
    const importName = dir.name.replace(/-/g, '_');
    return `import * as ${importName} from './${dir.name}';`;
  });

  const dirExports = directories.map((dir) => {
    const importName = dir.name.replace(/-/g, '_');
    return `export { ${importName} };`;
  });

  const indexContent = [...fileImports, ...dirImports, '', ...fileExports, ...dirExports, ''].join(
    '\n',
  );

  // Write the index.ts file
  await fs.writeFile(path.join(dir, 'index.ts'), indexContent, 'utf-8');
  console.log(`Created index.ts in ${dir}`);
};

export const snippetProcessor: Processor = {
  processFile: processSnippet,
  processDirectory: processDirectory,
};
