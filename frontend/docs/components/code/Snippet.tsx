import React from "react";
import { CodeBlock } from "./CodeBlock";
import { Snippet as SnippetType } from "@/lib/snips";

interface SnippetProps {
  src: SnippetType;
}


// This is a server component that will be rendered at build time
export const Snippet = ({ src }: SnippetProps) => {
  if (!src) {
    throw new Error("src is required");
  }

  return (<>hello</>
    // <CodeBlock
    //   source={{
    //     githubUrl: `https://github.com/hatchet-dev/hatchet/blob/main/${src.source}`,
    //     raw: src.content,
    //     language: src.language,
    //     props: {
    //       path: src.source
    //     }
    //   }}
    //   // target={src.source}
    // />
  );
};
