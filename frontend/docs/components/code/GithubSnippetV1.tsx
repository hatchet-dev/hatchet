"use client";

import React, { useState, useEffect } from "react";
import { CodeBlock } from "./CodeBlock";

interface GithubSnippetProps {
  src: string;
}

export const GithubSnippetV1 = ({ src }: GithubSnippetProps) => {
  const [content, setContent] = useState('');
  const [language, setLanguage] = useState('txt');
  const [question, filePath] = src.split(":");

  useEffect(() => {
    if (!src) {
      throw new Error("src is required");
    }

    // Dynamic import of the snippet module
    import(`@/lib/${filePath}`).then((module) => {
      setContent(module.content);
      setLanguage(module.language);
    }).catch((error) => {
      console.error('Error loading snippet:', error);
      setContent(`// Error loading snippet: ${error.message}`);
    });
  }, [src]);

  return (
    <CodeBlock
      source={{
        raw: content || '// Loading...',
        language
      }}
      target={question}
    />
  );
};
