import React from "react";
import { parseDocComments } from "./codeParser";
import { Src } from "./codeData";
import CodeStyleRender from "./CodeStyleRender";
import { Button } from "../ui/button";
import {
  CheckIcon,
  CopyIcon,
  FoldVertical,
  MoveUpRight,
  UnfoldVertical,
} from "lucide-react";

interface CodeRendererProps {
  source: Src;
  target?: string;
}

export const CodeBlock = ({ source, target }: CodeRendererProps) => {
  const [collapsed, setCollapsed] = React.useState(true);
  const [plainText, setPlainText] = React.useState(false);
  const [copied, setCopied] = React.useState(false);

  const parsed = parseDocComments(source.raw, target, collapsed);

  const canCollapse = source.raw.includes("// ...") || source.raw.includes("# ...")

  return (
    <>
      <div className="z-10 bg-background flex flex-row gap-4 justify-between items-center pl-2 mb-2">
        {source.githubUrl && (
            <a
              href={source.githubUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="text-xs text-gray-500 font-mono hover:underline"
          >
            {source.props?.path}
          </a>
        )}
        <div className="flex gap-2 justify-end">
        {canCollapse && (
          <Button
            variant="ghost"
              size="sm"
              onClick={() => setCollapsed(!collapsed)}
            >
              {collapsed ? (
                <>
                  <UnfoldVertical className="w-4 h-4 mr-2" />
                  Expand
                </>
              ) : (
                <>
                  <FoldVertical className="w-4 h-4 mr-2" />
                  Collapse
                </>
              )}
              </Button>
        )}
        <Button
          variant="outline"
          size="sm"
            onClick={() => {
              navigator.clipboard.writeText(parsed);
              setCopied(true);
              setTimeout(() => setCopied(false), 2000);
            }}
          >
            {copied ? (
              <>
                <CheckIcon className="w-4 h-4 mr-2" />
                Copied
              </>
            ) : (
              <>
                <CopyIcon className="w-4 h-4 mr-2" />
                Copy
              </>
            )}
          </Button>
        </div>
      </div>
      <div>
        {!plainText && (
          <CodeStyleRender
            parsed={parsed}
            language={source.language || "text"}
          />
        )}
        {/* plain text for SEO */}
        <pre
          style={{ display: plainText ? "block" : "none" }}
          aria-hidden="true"
        >
          {parsed}
        </pre>
      </div>

      <div className="flex flex-row mt-2 justify-between">
        <div className="flex gap-4">
          {source.githubUrl && <a href={source.githubUrl} target="_blank" rel="noopener noreferrer">
            <Button variant="outline" size="sm" className="flex flex-row gap-2">
              <svg
                height="16"
                width="16"
                viewBox="0 0 16 16"
                fill="currentColor"
              >
                <path d="M8 0C3.58 0 0 3.58 0 8c0 3.54 2.29 6.53 5.47 7.59.4.07.55-.17.55-.38 0-.19-.01-.82-.01-1.49-2.01.37-2.53-.49-2.69-.94-.09-.23-.48-.94-.82-1.13-.28-.15-.68-.52-.01-.53.63-.01 1.08.58 1.23.82.72 1.21 1.87.87 2.33.66.07-.52.28-.87.51-1.07-1.78-.2-3.64-.89-3.64-3.95 0-.87.31-1.59.82-2.15-.08-.2-.36-1.02.08-2.12 0 0 .67-.21 2.2.82.64-.18 1.32-.27 2-.27.68 0 1.36.09 2 .27 1.53-1.04 2.2-.82 2.2-.82.44 1.1.16 1.92.08 2.12.51.56.82 1.27.82 2.15 0 3.07-1.87 3.75-3.65 3.95.29.25.54.73.54 1.48 0 1.07-.01 1.93-.01 2.2 0 .21.15.46.55.38A8.013 8.013 0 0016 8c0-4.42-3.58-8-8-8z" />
              </svg>
              View Full Code Example on GitHub{" "}
              <MoveUpRight className="w-3 h-3" />
            </Button>
          </a>}
        </div>
        <div className="flex gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => {
              setCollapsed(false);
              setPlainText(!plainText);
            }}
          >
            {plainText ? "Code Highlighted" : "Plain Text"}
          </Button>
        </div>
      </div>
    </>
  );
};
