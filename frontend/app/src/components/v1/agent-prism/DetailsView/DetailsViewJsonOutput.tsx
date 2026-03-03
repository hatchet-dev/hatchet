import { type FC } from "react";
import JSONPretty from "react-json-pretty";
import colors from "tailwindcss/colors";

import { agentPrismPrefix } from "../theme";

export interface JsonViewerProps {
  content: string;
  id: string;
  className?: string;
}

export const DetailsViewJsonOutput: FC<JsonViewerProps> = ({
  content,
  id,
  className = "",
}) => {
  return (
    <JSONPretty
      booleanStyle={`color: ${colors.blue[800]};`}
      className={`overflow-x-hidden rounded-xl p-4 text-left ${className}`}
      data={content}
      id={`json-pretty-${id}`}
      keyStyle={`color: oklch(var(--${agentPrismPrefix}-code-key));`}
      mainStyle={`color: oklch(var(--${agentPrismPrefix}-code-base)); font-size: 12px; white-space: pre-wrap; word-wrap: break-word; overflow-wrap: break-word;`}
      stringStyle={`color: oklch(var(--${agentPrismPrefix}-code-string));`}
      valueStyle={`color: oklch(var(--${agentPrismPrefix}-code-number));`}
    />
  );
};
