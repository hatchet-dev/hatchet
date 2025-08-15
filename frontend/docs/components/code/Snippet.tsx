import React from "react";
import { CodeBlock } from "./CodeBlock";
import { type Snippet as SnippetType } from "@/lib/snippet";

export const Snippet = ({ snippet }: { snippet: SnippetType }) => {
  return (
    <CodeBlock
      source={{
        githubUrl: snippet.githubUrl,
        raw: snippet.content,
        language: snippet.language,
      }}
    />
  );
};
