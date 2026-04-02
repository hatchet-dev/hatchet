import { useEffect, useState } from "react";
import { CodeBlock } from "./code/CodeBlock";

/** Returns the MCP runtime endpoint URL based on current origin. */
function useMcpRuntimeUrl(): string {
  const [origin, setOrigin] = useState("https://docs.hatchet.run");
  useEffect(() => {
    setOrigin(window.location.origin);
  }, []);
  return `${origin}/api/mcp-runtime`;
}

/** Renders the MCP runtime endpoint URL as inline code. */
export function McpRuntimeUrl() {
  const url = useMcpRuntimeUrl();
  return <code>{url}</code>;
}

/** Renders the Cursor MCP config for the runtime server with token placeholder. */
export function CursorRuntimeMcpConfig() {
  const url = useMcpRuntimeUrl();
  const config = JSON.stringify(
    {
      "hatchet-runtime": {
        command: "npx",
        args: ["-y", "mcp-remote", url],
        env: {
          MCP_HEADER_AUTHORIZATION: "Bearer your-token",
        },
      },
    },
    null,
    2,
  );
  return <CodeBlock source={{ language: "json", raw: config }} />;
}

/** Renders the Claude Code mcp add command for the runtime server. */
export function ClaudeCodeRuntimeCommand() {
  const url = useMcpRuntimeUrl();
  return (
    <CodeBlock
      source={{
        language: "bash",
        raw: `claude mcp add --transport http --header "Authorization: Bearer your-token" hatchet-runtime ${url}`,
      }}
    />
  );
}
