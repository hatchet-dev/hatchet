import React from "react";
import codeblocks from "../codeblocks.json";

const CodeCell = ({ language, blockName, runnable = false }) => {
  const getCode = () => {
    console.log(codeblocks);
    if (codeblocks[language]) {
      const block = codeblocks[language].find((b) => b.blockName === blockName);
      if (block) {
        console.log(block.code);
        return block.code;
      }
    }
    return `Code block "${blockName}" not found for language ${language}`;
  };

  const code = getCode();

  return (
    <pre>
      <code
        className={`language-${language}`}
        dangerouslySetInnerHTML={{ __html: code }}
      />
    </pre>
  );
};

export default CodeCell;
