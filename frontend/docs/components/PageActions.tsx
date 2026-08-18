"use client";

import React, { useCallback, useState } from "react";
import posthog from "posthog-js";

const DEFAULT_ORIGIN = "https://docs.hatchet.run";

function safeBase64Encode(str: string): string {
  if (typeof btoa === "function") {
    return btoa(str);
  }
  if (typeof Buffer !== "undefined") {
    return Buffer.from(str).toString("base64");
  }
  return "";
}

import { RSS_FEED_PATHS } from "@/lib/rss-feeds";

const CursorIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
    <path d="M22.106 5.68L12.5.135a.998.998 0 00-.998 0L1.893 5.68a.84.84 0 00-.419.726v11.186c0 .3.16.577.42.727l9.607 5.547a.999.999 0 00.998 0l9.608-5.547a.84.84 0 00.42-.727V6.407a.84.84 0 00-.42-.726zm-.603 1.176L12.228 22.92c-.063.108-.228.064-.228-.061V12.34a.59.59 0 00-.295-.51l-9.11-5.26c-.107-.062-.063-.228.062-.228h18.55c.264 0 .428.286.296.514z" />
  </svg>
);

const ClaudeIcon = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="currentColor">
    <path d="M4.709 15.955l4.72-2.647.08-.23-.08-.128H9.2l-.79-.048-2.698-.073-2.339-.097-2.266-.122-.571-.121L0 11.784l.055-.352.48-.321.686.06 1.52.103 2.278.158 1.652.097 2.449.255h.389l.055-.157-.134-.098-.103-.097-2.358-1.596-2.552-1.688-1.336-.972-.724-.491-.364-.462-.158-1.008.656-.722.881.06.225.061.893.686 1.908 1.476 2.491 1.833.365.304.145-.103.019-.073-.164-.274-1.355-2.446-1.446-2.49-.644-1.032-.17-.619a2.97 2.97 0 01-.104-.729L6.283.134 6.696 0l.996.134.42.364.62 1.414 1.002 2.229 1.555 3.03.456.898.243.832.091.255h.158V9.01l.128-1.706.237-2.095.23-2.695.08-.76.376-.91.747-.492.584.28.48.685-.067.444-.286 1.851-.559 2.903-.364 1.942h.212l.243-.242.985-1.306 1.652-2.064.73-.82.85-.904.547-.431h1.033l.76 1.129-.34 1.166-1.064 1.347-.881 1.142-1.264 1.7-.79 1.36.073.11.188-.02 2.856-.606 1.543-.28 1.841-.315.833.388.091.395-.328.807-1.969.486-2.309.462-3.439.813-.042.03.049.061 1.549.146.662.036h1.622l3.02.225.79.522.474.638-.079.485-1.215.62-1.64-.389-3.829-.91-1.312-.329h-.182v.11l1.093 1.068 2.006 1.81 2.509 2.33.127.578-.322.455-.34-.049-2.205-1.657-.851-.747-1.926-1.62h-.128v.17l.444.649 2.345 3.521.122 1.08-.17.353-.608.213-.668-.122-1.374-1.925-1.415-2.167-1.143-1.943-.14.08-.674 7.254-.316.37-.729.28-.607-.461-.322-.747.322-1.476.389-1.924.315-1.53.286-1.9.17-.632-.012-.042-.14.018-1.434 1.967-2.18 2.945-1.726 1.845-.414.164-.717-.37.067-.662.401-.589 2.388-3.036 1.44-1.882.93-1.086-.006-.158h-.055L4.132 18.56l-1.13.146-.487-.456.061-.746.231-.243 1.908-1.312-.006.006z" />
  </svg>
);

const MarkdownIcon = () => (
  <svg
    width="12"
    height="12"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
    <polyline points="14 2 14 8 20 8" />
    <line x1="16" y1="13" x2="8" y2="13" />
    <line x1="16" y1="17" x2="8" y2="17" />
    <polyline points="10 9 9 9 8 9" />
  </svg>
);

const RssIcon = () => (
  <svg
    width="12"
    height="12"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <path d="M4 11a9 9 0 0 1 9 9" />
    <path d="M4 4a16 16 0 0 1 16 16" />
    <circle cx="5" cy="19" r="1" />
  </svg>
);

const CopyIcon = () => (
  <svg
    width="12"
    height="12"
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    strokeWidth="2"
    strokeLinecap="round"
    strokeLinejoin="round"
  >
    <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
    <path d="M5 15H4a2 2 0 01-2-2V4a2 2 0 012-2h9a2 2 0 012 2v1" />
  </svg>
);

const pageLinkStyle: React.CSSProperties = {
  fontSize: "0.75rem",
  opacity: 0.5,
  textDecoration: "none",
  display: "inline-flex",
  alignItems: "center",
  gap: "4px",
  cursor: "pointer",
};

function CopyClaudeButton({ command }: { command: string }) {
  const [copied, setCopied] = useState(false);

  const handleClick = useCallback(
    (e: React.MouseEvent) => {
      e.preventDefault();
      navigator.clipboard.writeText(command);
      setCopied(true);
      posthog.capture("mcp_install_click", {
        editor: "claude-code",
        method: "copy_command",
        page: window.location.pathname,
      });
      setTimeout(() => setCopied(false), 1500);
    },
    [command],
  );

  return (
    <a
      href="#"
      onClick={handleClick}
      style={pageLinkStyle}
      title="Add to Claude"
    >
      <ClaudeIcon />
      <span className="page-action-label">
        {copied ? "Copied! Run in terminal" : "Add to Claude"}
      </span>
    </a>
  );
}

function CopyForLLMButton({
  markdownHref,
  pathname,
}: {
  markdownHref: string;
  pathname: string;
}) {
  const [copied, setCopied] = useState(false);
  const [loading, setLoading] = useState(false);

  const handleClick = useCallback(
    async (e: React.MouseEvent) => {
      e.preventDefault();
      if (loading || copied) return;
      setLoading(true);
      try {
        const base =
          typeof window !== "undefined"
            ? window.location.origin
            : DEFAULT_ORIGIN;
        const res = await fetch(`${base}${markdownHref}`);
        if (!res.ok) throw new Error("Failed to fetch");
        const text = await res.text();
        await navigator.clipboard.writeText(text);
        setCopied(true);
        posthog.capture("docs_copy_for_llm", { page: pathname });
        setTimeout(() => setCopied(false), 2000);
      } catch {
        // no-op
      } finally {
        setLoading(false);
      }
    },
    [markdownHref, pathname, loading, copied],
  );

  return (
    <a
      href="#"
      onClick={handleClick}
      style={pageLinkStyle}
      title="Copy page as Markdown for LLM"
    >
      <CopyIcon />
      <span className="page-action-label">
        {loading ? "..." : copied ? "Copied!" : "Copy for LLM"}
      </span>
    </a>
  );
}

export function PageActions({ pathname }: { pathname: string }) {
  const [origin] = useState(() =>
    typeof window !== "undefined" ? window.location.origin : DEFAULT_ORIGIN,
  );

  const normalized = pathname.replace(/^\//, "").replace(/\/$/, "") || "index";
  const llmsMarkdownHref = `/llms/${normalized}.md`;

  const hasRssFeed = RSS_FEED_PATHS.includes(normalized);
  const rssFeedHref = hasRssFeed ? `/${normalized}/feed.xml` : "";

  const mcpUrl = `${origin}/api/mcp`;
  const cursorConfig = JSON.stringify({
    command: "npx",
    args: ["-y", "mcp-remote", mcpUrl],
  });
  const cursorDeeplink = `cursor://anysphere.cursor-deeplink/mcp/install?name=hatchet-docs&config=${safeBase64Encode(cursorConfig)}`;

  const claudeCommand = `claude mcp add --transport http hatchet-docs ${mcpUrl}`;

  return (
    <div className="page-actions">
      <a
        href={cursorDeeplink}
        style={pageLinkStyle}
        onClick={() =>
          posthog.capture("mcp_install_click", {
            editor: "cursor",
            method: "deeplink",
            page: normalized,
          })
        }
        title="Add to Cursor"
      >
        <CursorIcon />
        <span className="page-action-label">Add to Cursor</span>
      </a>
      <CopyClaudeButton command={claudeCommand} />
      <CopyForLLMButton markdownHref={llmsMarkdownHref} pathname={normalized} />
      <a
        href={llmsMarkdownHref}
        target="_blank"
        rel="noopener noreferrer"
        style={pageLinkStyle}
        onClick={() =>
          posthog.capture("docs_view_markdown", { page: normalized })
        }
        title="View as Markdown"
      >
        <MarkdownIcon />
        <span className="page-action-label">View as MD</span>
      </a>
      {hasRssFeed && (
        <a
          href={rssFeedHref}
          target="_blank"
          rel="noopener noreferrer"
          style={pageLinkStyle}
          onClick={() =>
            posthog.capture("docs_rss_click", {
              feed: rssFeedHref,
              page: normalized,
            })
          }
          title="Subscribe via RSS"
        >
          <RssIcon />
          <span className="page-action-label">RSS</span>
        </a>
      )}
    </div>
  );
}
