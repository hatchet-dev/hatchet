import React from "react";
import { CodeBlock } from "./CodeBlock";
import { Snippet as SnippetType } from "@/lib/snips";

interface SnippetProps {
  src: SnippetType;
  block?: keyof SnippetType['blocks'] | 'ALL';
}


const languageMap = {
  typescript: 'typescript',
  python: 'py',
  go: 'go',
  unknown: 'unknown',
};

// This is a server component that will be rendered at build time
export const Snippet = ({ src, block }: SnippetProps) => {
  if (!src || !src.content) {
    throw new Error("src is required");
  }

  let content = src.content;

  if (block && block !== 'ALL' && src.blocks) {
    if (!(block in src.blocks)) {
      throw new Error(`Block ${block} not found in ${src.source} ${JSON.stringify(src.blocks, null, 2)}`);
    }

    const lines = src.content.split('\n');
    content = lines.slice(src.blocks[block].start - 1, src.blocks[block].stop).join('\n');
  }


  return (
    <>
    <CodeBlock
      source={{
        githubUrl: `https://github.com/hatchet-dev/hatchet/blob/main/${src.source}`,
        raw: content || '',
        language: languageMap[src.language as keyof typeof languageMap],
        props: {
          path: src.source
        }
      }}
    />
    </>
  );
};
