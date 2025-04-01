import React from "react";
import { useData } from "nextra/hooks";
import { CodeBlock } from "./CodeBlock";
import { RepoProps, Src } from "./codeData";

interface GithubSnippetProps {
  src: RepoProps;
  target: string;
}

export const GithubSnippet = ({ src, target }: GithubSnippetProps) => {
  const { contents } = useData();
  const snippet = contents.find((c) => c.rawUrl.endsWith(src.path)) as Src;

  if (!snippet) {
    return null;
  }

  return (
    <CodeBlock
      source={{
        ...src,
        ...snippet,
      }}
      target={target}
    />
  );
};
