import React, { useEffect, useState } from "react";
import { codeToHtml } from "shiki";
import codeblocks from "../codeblocks.json";

const CodeCell = ({ language, blockName, runnable = false }) => {
  const [html, setHtml] = useState("");

  useEffect(() => {
    const fetchCode = async () => {
      const getCode = () => {
        if (codeblocks[language]) {
          const block = codeblocks[language].find(
            (b) => b.blockName === blockName,
          );
          if (block) {
            return block.code;
          }
        }
        return `Code block "${blockName}" not found for language ${language}`;
      };

      const code = getCode();
      const highlightedHtml = await codeToHtml(code, {
        lang: language.toLowerCase(),
        theme: "github-dark",
      });
      setHtml(highlightedHtml);
    };

    fetchCode();
  }, [language, blockName]);

  console.log(html);

  return (
    <div dangerouslySetInnerHTML={{ __html: html }}></div>
  );
};

export default CodeCell;
