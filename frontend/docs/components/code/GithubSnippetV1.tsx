import React from "react";
import { CodeBlock } from "./CodeBlock";
import snips, { snippets } from "@/lib/snips";

interface GithubSnippetProps {
  src: string;
}



// This is a server component that will be rendered at build time
export const GithubSnippetV1 = ({ src }: GithubSnippetProps) => {
  if (!src) {
    throw new Error("src is required");
  }

  const [question, filePath] = src.split(":");
  
  
  // Get the snippet content from the snippets object
  const snippet = snippets[filePath];
  if (!snippet) {
    throw new Error(`Snippet content not found: ${filePath}`);
  }
  
  return (
    <CodeBlock
      source={{
        raw: snippet.content,
        language: snippet.language
      }}
      target={question}
    />
  );
};
