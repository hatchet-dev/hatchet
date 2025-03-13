"use client";

import { useTheme } from "nextra-theme-docs";
import { useEffect, useState, useMemo } from "react";
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
      const highlightedHtml = await codeToHtml(parsed, {
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
