import { cn } from "@/lib/utils";
import { Light as SyntaxHighlighter } from "react-syntax-highlighter";

import typescript from "react-syntax-highlighter/dist/esm/languages/hljs/typescript";
import yaml from "react-syntax-highlighter/dist/esm/languages/hljs/yaml";
import json from "react-syntax-highlighter/dist/esm/languages/hljs/json";

import { anOldHope } from "react-syntax-highlighter/dist/esm/styles/hljs";

SyntaxHighlighter.registerLanguage("typescript", typescript);
SyntaxHighlighter.registerLanguage("yaml", yaml);
SyntaxHighlighter.registerLanguage("json", json);

export function Code({
  children,
  language,
  className,
  maxHeight,
}: {
  children: string;
  language: string;
  className?: string;
  maxHeight?: string;
}) {
  return (
    <div className={cn("text-xs", className)}>
      <SyntaxHighlighter
        children={children.trim()}
        language={language}
        style={anOldHope}
        customStyle={{
          background: "hsl(var(--muted) / 0.5)",
          borderRadius: "0.5rem",
          maxHeight: maxHeight,
          fontFamily:
            "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', 'Courier New', monospace",
        }}
      />
    </div>
  );
}

// import { Fragment } from "react";
// import { Highlight, themes } from "prism-react-renderer";

// const theme = themes.nightOwl;

// export function Code({
//   children,
//   language,
// }: {
//   children: string;
//   language: string;
// }) {
//   return (
//     <Highlight code={children.trimEnd()} language={language} theme={theme}>
//       {({ className, style, tokens, getTokenProps }) => (
//         <pre className={className} style={style}>
//           <code>
//             {tokens.map((line, lineIndex) => (
//               <Fragment key={lineIndex}>
//                 {line
//                   .filter((token) => !token.empty)
//                   .map((token, tokenIndex) => (
//                     <span key={tokenIndex} {...getTokenProps({ token })} />
//                   ))}
//                 {"\n"}
//               </Fragment>
//             ))}
//           </code>
//         </pre>
//       )}
//     </Highlight>
//   );
// }
