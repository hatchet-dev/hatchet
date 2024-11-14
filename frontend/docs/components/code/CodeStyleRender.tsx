"use client";

import { useTheme } from "nextra-theme-docs";
import React, { useEffect, useState } from "react";
import { codeToHtml } from "shiki";

interface CodeStyleRenderProps {
  parsed: string
  language: string
}

const CodeStyleRender = ({ parsed, language }: CodeStyleRenderProps) => {
  const [html, setHtml] = useState<string>("");
  const theme = useTheme();


  useEffect(() => {
    const asyncHighlight = async () => {
        const highlightedHtml = await codeToHtml(parsed, {
            lang: language.toLowerCase(),
            theme: theme.theme === "dark" ? "github-dark" : "github-light",
        });

        setHtml(highlightedHtml);
    }

    asyncHighlight();
  }, [parsed, language, theme.theme]);

  return <div dangerouslySetInnerHTML={{ __html: html }}></div>
};

export default CodeStyleRender;
