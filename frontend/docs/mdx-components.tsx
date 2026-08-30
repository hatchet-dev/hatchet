import defaultMdxComponents from "fumadocs-ui/mdx";
import {
  Callout,
  Cards,
  Steps,
  Tabs,
  FileTree,
} from "@/components/nextra-compat";
import { Mermaid } from "@/components/Mermaid";

export function getMDXComponents(components: Record<string, unknown> = {}) {
  return {
    ...defaultMdxComponents,
    Callout,
    Cards,
    Steps,
    Tabs,
    FileTree,
    Mermaid,
    ...components,
  };
}
