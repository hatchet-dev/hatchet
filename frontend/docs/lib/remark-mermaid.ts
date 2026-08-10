import { visit } from "unist-util-visit";
import type { Root, Code } from "mdast";

export function remarkMermaid() {
  return (tree: Root) => {
    visit(tree, "code", (node: Code, index, parent) => {
      if (node.lang !== "mermaid" || !parent || index === undefined) return;
      parent.children[index] = {
        type: "mdxJsxFlowElement",
        name: "Mermaid",
        attributes: [
          { type: "mdxJsxAttribute", name: "chart", value: node.value },
        ],
        children: [],
      } as never;
    });
  };
}
