import React from "react";
import { CodeBlock } from "./CodeBlock";
import { type Snippet as SnippetType } from "@/lib/snippet";

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
