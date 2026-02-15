import { useEffect, useState } from "react";
import { CodeBlock } from "./code/CodeBlock";

/* ── Tab label styles ─────────────────────────────────────── */

const tabLabelStyle: React.CSSProperties = {
  display: "inline-flex",
  alignItems: "center",
  gap: "6px",
};

/** Renders an SVG as a CSS mask filled with currentColor (works in light + dark mode). */
function ThemedIcon({ src }: { src: string }) {
  return (
    <span
      style={{
        display: "inline-block",
        width: 16,
        height: 16,
        flexShrink: 0,
        backgroundColor: "currentColor",
        WebkitMaskImage: `url(${src})`,
        WebkitMaskSize: "contain",
        WebkitMaskRepeat: "no-repeat",
        WebkitMaskPosition: "center",
        maskImage: `url(${src})`,
        maskSize: "contain",
        maskRepeat: "no-repeat",
        maskPosition: "center",
      } as React.CSSProperties}
    />
  );
}

/** Cursor IDE tab label with official logo. */
export function CursorTabLabel() {
  return (
    <span style={tabLabelStyle}>
      <ThemedIcon src="/cursor-logo.svg" />
      Cursor
    </span>
  );
}

/** Claude Code tab label with official Claude logo. */
export function ClaudeCodeTabLabel() {
  return (
    <span style={tabLabelStyle}>
      <ThemedIcon src="/claude-logo.svg" />
      Claude Code
    </span>
  );
}

/** Claude Desktop tab label with official Claude logo. */
export function ClaudeDesktopTabLabel() {
  return (
    <span style={tabLabelStyle}>
      <ThemedIcon src="/claude-logo.svg" />
      Claude Desktop
    </span>
  );
}

/** Globe icon – used for the "Other Agents" tab. */
export function OtherAgentsTabLabel() {
  return (
    <span style={tabLabelStyle}>
      <svg
        width="16"
        height="16"
        viewBox="0 0 24 24"
        fill="none"
        stroke="currentColor"
        strokeWidth="2"
        strokeLinecap="round"
        strokeLinejoin="round"
      >
        <circle cx="12" cy="12" r="10" />
        <path d="M2 12h20" />
        <path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10A15.3 15.3 0 0 1 12 2z" />
      </svg>
      Other Agents
    </span>
  );
}

/** Returns the MCP endpoint URL based on current origin. */
function useMcpUrl(): string {
  const [origin, setOrigin] = useState("https://docs.hatchet.run");
  useEffect(() => {
    setOrigin(window.location.origin);
  }, []);
  return `${origin}/api/mcp`;
}

/** Renders the MCP endpoint URL as inline code. */
export function McpUrl() {
  const url = useMcpUrl();
  return <code>{url}</code>;
}

/** Cursor one-click install deeplink button. */
export function CursorDeeplinkButton() {
  const url = useMcpUrl();
  const config = JSON.stringify({
    command: "npx",
    args: ["-y", "mcp-remote", url],
  });
  const encoded =
    typeof window !== "undefined"
      ? btoa(config)
      : Buffer.from(config).toString("base64");
  const deeplink = `cursor://anysphere.cursor-deeplink/mcp/install?name=hatchet-docs&config=${encoded}`;

  return (
    <a
      href={deeplink}
      style={{
        display: "inline-block",
        padding: "8px 16px",
        background: "#3392FF",
        color: "white",
        borderRadius: "8px",
        fontWeight: 600,
        textDecoration: "none",
        fontSize: "14px",
      }}
    >
      Install MCP Server in Cursor
    </a>
  );
}

/** Renders a JSON config code block with the dynamic MCP URL. */
export function CursorMcpConfig() {
  const url = useMcpUrl();
  const config = JSON.stringify(
    { "hatchet-docs": { command: "npx", args: ["-y", "mcp-remote", url] } },
    null,
    2,
  );
  return <CodeBlock source={{ language: "json", raw: config }} />;
}

/** Renders the claude mcp add command with dynamic URL. */
export function ClaudeCodeCommand() {
  const url = useMcpUrl();
  return (
    <CodeBlock
      source={{
        language: "bash",
        raw: `claude mcp add --transport http hatchet-docs ${url}`,
      }}
    />
  );
}

/** Renders Claude Desktop JSON config with dynamic URL. */
export function ClaudeDesktopConfig() {
  const url = useMcpUrl();
  const config = JSON.stringify(
    {
      mcpServers: {
        "hatchet-docs": { command: "npx", args: ["-y", "mcp-remote", url] },
      },
    },
    null,
    2,
  );
  return <CodeBlock source={{ language: "json", raw: config }} />;
}
