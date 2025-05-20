"use client";

import { useTheme } from "nextra-theme-docs";
import React, { useEffect, useState, useMemo } from "react";
import { codeToHtml } from "shiki";

interface CodeStyleRenderProps {
  parsed: string;
  language: string;
}

const CodeStyleRender = ({ parsed, language }: CodeStyleRenderProps) => {
  const [html, setHtml] = useState<string>("");
  const theme = useTheme();

  const themeName = useMemo(() => {
    return theme.resolvedTheme === "dark" ? "github-dark" : "github-light";
  }, [theme.resolvedTheme]);

  useEffect(() => {
    const asyncHighlight = async () => {
      const dedentedCode = dedent(parsed);
      const highlightedHtml = await codeToHtml(dedentedCode, {
        lang: language.toLowerCase(),
        theme: themeName,
      });

      setHtml(highlightedHtml);
    };

    asyncHighlight();
  }, [parsed, language, themeName]);

  return (
    <>
      <div dangerouslySetInnerHTML={{ __html: html }}></div>
    </>
  );
};

export default CodeStyleRender;

function dedent(code: string) {
  const lines = code.split("\n");
  const nonEmptyLines = lines.filter((line) => line.trim().length > 0);

  if (nonEmptyLines.length === 0) {
    return code;
  }

  const minIndent = Math.min(
    ...nonEmptyLines.map((line) => {
      const match = line.match(/^(\s*)/);
      return match ? match[1].length : 0;
    })
  );

  if (minIndent > 0) {
    return lines
      .map((line) => {
        if (line.trim().length > 0) {
          return line.slice(minIndent);
        }

        return line;
      })
      .join("\n");
  }

  return code;
}
