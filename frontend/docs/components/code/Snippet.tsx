import React from "react";
import { CodeBlock } from "./CodeBlock";
import { type Snippet as SnippetType } from "@/lib/snippet";

type Language = SnippetType["language"];

// See the list of supported languages for how to define these:
// https://highlightjs.readthedocs.io/en/latest/supported-languages.html
const languageToHighlightAbbreviation = (language: Language) => {
  switch (language) {
    case "python":
      return "py";
    case "typescript":
      return "ts";
    case "go":
      return "go";
    case "ruby":
      return "rb";
    default:
      const exhaustiveCheck: never = language;
      throw new Error(`Unsupported language: ${exhaustiveCheck}`);
  }
};

export const Snippet = ({ src }: { src: SnippetType }) => {
  if (src === undefined) {
    throw new Error(
      "Snippet was undefined. You probably provided a path to a snippet that doesn't exist.",
    );
  }
  return (
    <CodeBlock
      source={{
        githubUrl: src.githubUrl,
        raw: src.content,
        language: languageToHighlightAbbreviation(src.language),
        codePath: src.codePath,
      }}
    />
  );
};
