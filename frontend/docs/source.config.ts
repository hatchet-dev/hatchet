import {
  defineDocs,
  defineConfig,
  frontmatterSchema,
} from "fumadocs-mdx/config";
import { z } from "zod";
import { remarkMermaid } from "./lib/remark-mermaid";

export const docs = defineDocs({
  dir: "content/docs",
  docs: {
    // cast: zod's .extend() blows TS instantiation depth against fumadocs' schema types
    schema: frontmatterSchema.extend({
      seoTitle: z.string().optional(),
    }) as unknown as typeof frontmatterSchema,
  },
});

export default defineConfig({
  mdxOptions: {
    remarkPlugins: [remarkMermaid],
    rehypeCodeOptions: {
      themes: {
        light: "github-light",
        dark: "github-dark",
      },
      fallbackLanguage: "txt",
    },
  },
});
