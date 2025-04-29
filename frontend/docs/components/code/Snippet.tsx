import React, { useMemo } from "react";
import { CodeBlock } from "./CodeBlock";
import { Snippet as SnippetType } from "@/lib/snips";

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

// This is a server component that will be rendered at build time
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
        source={{
          githubUrl: `https://github.com/hatchet-dev/hatchet/blob/main/${fixedSource}`,
          raw: content || '',
          language: language,
          props: {
            path: fixedSource,
          },
        }}
    />
    </>
  );
};
