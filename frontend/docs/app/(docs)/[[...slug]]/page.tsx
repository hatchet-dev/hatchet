import { source } from "@/lib/source";
import { DocsPage, DocsBody, DocsTitle } from "fumadocs-ui/page";
import defaultMdxComponents from "fumadocs-ui/mdx";
import { notFound } from "next/navigation";
import {
  Children,
  isValidElement,
  type ComponentProps,
  type ReactNode,
} from "react";
import { getMDXComponents } from "@/mdx-components";
import { PageActions } from "@/components/PageActions";
import { RSS_FEED_PATHS } from "@/lib/rss-feeds";
import { getLastModified } from "@/lib/last-modified";
import {
  headingNamesSdk,
  pageDescription,
  pageTitle,
  sdkLabel,
  type SeoPageData,
} from "@/lib/seo";

const DefaultH1 = defaultMdxComponents.h1;

function textOf(node: ReactNode): string {
  if (typeof node === "string" || typeof node === "number") return String(node);
  if (Array.isArray(node)) return node.map(textOf).join("");
  if (isValidElement<{ children?: ReactNode }>(node)) {
    return Children.toArray(node.props.children).map(textOf).join("");
  }
  return "";
}

function SdkBadge({ label }: { label: string }) {
  return (
    <span className="ms-3 inline-block rounded-md border border-fd-border px-2 py-0.5 align-middle text-sm font-medium text-fd-muted-foreground">
      {label}
    </span>
  );
}

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

  // The SDK reference is generated once per language, so the same heading
  // ("Cron Client", "Context", ...) appears on four pages. Tag the H1 with the
  // language so each page has a distinct heading, unless the heading already
  // names it ("Hatchet Python SDK Reference").
  const sdk = sdkLabel(slug);
  // Most pages carry their own `# Heading`; generated reference pages,
  // changelogs and a few guides start at H2, so render the title for them.
  const hasBodyH1 = page.data.toc.some((item) => item.depth === 1);
  const components = getMDXComponents(
    sdk
      ? {
          h1: ({ children, ...props }: ComponentProps<typeof DefaultH1>) => (
            <DefaultH1 {...props}>
              {children}
              {!headingNamesSdk(textOf(children), sdk) && (
                <SdkBadge label={sdk} />
              )}
            </DefaultH1>
          ),
        }
      : {},
  );

  return (
    <DocsPage
      toc={page.data.toc}
      tableOfContent={{ enabled: !noToc, style: "clerk" }}
      full={noToc}
    >
      <PageActions pathname={"/" + slug.join("/")} />
      {!hasBodyH1 && (
        <DocsTitle>
          {(page.data as SeoPageData).seoTitle ?? page.data.title}
          {sdk &&
            !headingNamesSdk(
              (page.data as SeoPageData).seoTitle ?? page.data.title,
              sdk,
            ) && <SdkBadge label={sdk} />}
        </DocsTitle>
      )}
      <DocsBody>
        <MDX components={components} />
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
  const data = page.data as SeoPageData;
  const title = pageTitle(slug, data);
  const description = await pageDescription(data, sdkLabel(slug));
  const canonical = `/${pathname}`;

  return {
    title,
    description,
    alternates: {
      canonical,
      types: {
        "text/markdown": `/llms/${pathname}.md`,
        ...(RSS_FEED_PATHS.includes(pathname)
          ? { "application/rss+xml": `/${pathname}/feed.xml` }
          : {}),
      },
    },
    openGraph: {
      title: `${title} - Hatchet Documentation`,
      description,
      url: canonical,
      siteName: "Hatchet Documentation",
      type: "article",
      images: ["/og.png"],
    },
    twitter: {
      card: "summary_large_image",
      title: `${title} - Hatchet Documentation`,
      description,
      images: ["/og.png"],
    },
  };
}
