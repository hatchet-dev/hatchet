import { source } from "@/lib/source";
import { DocsPage, DocsBody } from "fumadocs-ui/page";
import { notFound } from "next/navigation";
import { getMDXComponents } from "@/mdx-components";
import { PageActions } from "@/components/PageActions";
import { RSS_FEED_PATHS } from "@/lib/rss-feeds";
import { getLastModified } from "@/lib/last-modified";

export default async function Page({
  params,
}: {
  params: Promise<{ slug?: string[] }>;
}) {
  const { slug } = await params;
  if (!slug?.length) notFound();
  const page = source.getPage(slug);
  if (!page) notFound();

  const MDX = page.data.body;
  const noToc = slug[0] === "agent-instructions";
  const lastModified = getLastModified(slug);
  const lastUpdatedLabel =
    lastModified !== null
      ? new Date(lastModified).toLocaleDateString("en-US", {
          year: "numeric",
          month: "long",
          day: "numeric",
        })
      : null;

  return (
    <DocsPage
      toc={page.data.toc}
      tableOfContent={{ enabled: !noToc, style: "clerk" }}
      full={noToc}
    >
      <PageActions pathname={"/" + slug.join("/")} />
      <DocsBody>
        <MDX components={getMDXComponents()} />
      </DocsBody>
      {lastUpdatedLabel && (
        <p className="mt-8 text-sm text-fd-muted-foreground">
          Last updated on {lastUpdatedLabel}
        </p>
      )}
    </DocsPage>
  );
}

export function generateStaticParams() {
  return source.generateParams();
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ slug?: string[] }>;
}) {
  const { slug } = await params;
  if (!slug?.length) return {};
  const page = source.getPage(slug);
  if (!page) return {};

  const pathname = slug.join("/");
  const seoTitle = (page.data as { seoTitle?: string }).seoTitle;

  return {
    title: seoTitle ?? page.data.title,
    description: page.data.description,
    alternates: {
      types: {
        "text/markdown": `/llms/${pathname}.md`,
        ...(RSS_FEED_PATHS.includes(pathname)
          ? { "application/rss+xml": `/${pathname}/feed.xml` }
          : {}),
      },
    },
  };
}
