import type { MetadataRoute } from "next";
import { isHiddenPage } from "@/lib/hidden-pages";
import { source } from "@/lib/source";

export default function sitemap(): MetadataRoute.Sitemap {
  return source
    .getPages()
    .filter((page) => !isHiddenPage(page.url))
    .map((page) => ({ url: `https://docs.hatchet.run${page.url}` }));
}
