import React from "react";
import { parseDocComments } from "./codeParser";
import { Src } from "./codeData";

interface CodeRendererProps {
  source: Src;
  target: string;
}

export const CodeRenderer = ({ source, target }: CodeRendererProps) => {
    const [collapsed, setCollapsed] = React.useState(true);
    const [plainText, setPlainText] = React.useState(false);


    const parsed = parseDocComments(source.raw, target, collapsed);

    return <>
        <button onClick={() => setCollapsed(!collapsed)}>
            {collapsed ? 'Expand' : 'Collapse'}
        </button>
        <button onClick={() => setPlainText(!plainText)}>
            {plainText ? 'Show HTML' : 'Show Plain Text'}
        </button>
        {
            plainText ? <pre>{parsed}</pre> :
                <pre>{parsed}</pre>
        }
    </>
}
