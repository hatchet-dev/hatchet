// MDX components for Nextra 4
import { Callout, Cards, Steps, Tabs, FileTree } from "nextra/components";

export function useMDXComponents(components: Record<string, unknown>) {
  return {
    ...components,
    // Adding Nextra components so they can be used in MDX files
    Callout,
    Cards,
    Steps,
    Tabs,
    FileTree,
  };
}
