import React from "react";
import { CodeBlock } from "./CodeBlock";
import { type Snippet as SnippetType } from "@/lib/snippet";

export const Snippet = ({ src }: { src: SnippetType }) => {
  return (
    <CodeBlock
      source={{
        githubUrl: src.githubUrl,
        raw: src.content,
        language: src.language,
      }}
    />
  );
};
