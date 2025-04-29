import { useMemo } from 'react';
import { CodeBlock } from './code-block';
import { Snippet as SnippetType } from '@/next/lib/docs/snips';
interface SnippetProps {
  src: SnippetType;
  block?: keyof SnippetType['blocks'] | 'ALL';
}

const languageMap = {
  typescript: 'ts',
  python: 'py',
  go: 'go',
  unknown: 'txt',
};

// This will be rendered at build time
export const Snippet = ({ src, block }: SnippetProps) => {
  if (!src.content) {
    throw new Error(`src content is required: ${src.source}`);
  }

  const language = useMemo(() => {
    const normalizedLanguage = src.language?.toLowerCase().trim();
    if (normalizedLanguage && normalizedLanguage in languageMap) {
      return languageMap[normalizedLanguage as keyof typeof languageMap];
    }
    return 'txt';
  }, [src.language]);

  let content = src.content;

  if (block && block !== 'ALL' && src.blocks) {
    if (!(block in src.blocks)) {
      throw new Error(
        `Block ${block} not found in ${src.source} ${JSON.stringify(src.blocks, null, 2)}`,
      );
    }

    const lines = src.content.split('\n');
    content = lines
      .slice(src.blocks[block].start - 1, src.blocks[block].stop)
      .join('\n');
  }

  const fixedSource = src.source.replace('out/', 'examples/');

  return (
    <>
      <CodeBlock
        value={content}
        language={language}
        // highlightLines={src.blocks?.[block]?.start}
        // highlightLines={src.blocks?.[block]?.stop}
        title={fixedSource}
        link={`https://github.com/hatchet-dev/hatchet/blob/main/${fixedSource}`}
      />
    </>
  );
};
