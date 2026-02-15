import React, { useCallback, useEffect, useState } from "react";
import { useConfig, useTheme } from "nextra-theme-docs";
import { useRouter } from "next/router";
import posthog from "posthog-js";
import Search from "@/components/Search";

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
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M14 2H6a2 2 0 00-2 2v16a2 2 0 002 2h12a2 2 0 002-2V8z" />
    <polyline points="14 2 14 8 20 8" />
    <line x1="16" y1="13" x2="8" y2="13" />
    <line x1="16" y1="17" x2="8" y2="17" />
    <polyline points="10 9 9 9 8 9" />
  </svg>
);

function CopyClaudeButton({ command }: { command: string }) {
  const [copied, setCopied] = useState(false);

  const handleClick = useCallback((e: React.MouseEvent) => {
    e.preventDefault();
    navigator.clipboard.writeText(command);
    setCopied(true);
    posthog.capture("mcp_install_click", {
      editor: "claude-code",
      method: "copy_command",
      page: window.location.pathname,
    });
    setTimeout(() => setCopied(false), 1500);
  }, [command]);

  return (
    <a href="#" onClick={handleClick} style={pageLinkStyle} title="Add to Claude">
      <ClaudeIcon />
      <span className="page-action-label">{copied ? "Copied! Run in terminal" : "Add to Claude"}</span>
    </a>
  );
}

const pageLinkStyle: React.CSSProperties = {
  fontSize: "0.75rem",
  opacity: 0.5,
  textDecoration: "none",
  display: "inline-flex",
  alignItems: "center",
  gap: "4px",
  cursor: "pointer",
};

const config = {
  logo: (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 198 49"
      width="120"
      height="35"
      fill="none"
      // preserveAspectRatio="xMidYMid meet"
    >
      <path
        fill="var(--brand)"
        d="m25.22 21.618 6.486-14.412 12.402 12.402c1.713 1.713 2.569 2.569 2.89 3.556a4.324 4.324 0 0 1 0 2.672c-.321.987-1.177 1.843-2.89 3.556L31.706 41.794H21.618l16.936-18.239c.568-.612.852-.918.866-1.179a.72.72 0 0 0-.258-.59c-.2-.168-.618-.168-1.453-.168H25.221ZM23.78 27.382l-6.486 14.412L4.892 29.392c-1.713-1.713-2.569-2.569-2.89-3.556a4.324 4.324 0 0 1 0-2.672c.321-.987 1.177-1.843 2.89-3.556L17.294 7.207h10.088L10.447 25.445c-.568.612-.852.918-.866 1.179a.72.72 0 0 0 .258.59c.2.168.618.168 1.453.168H23.78ZM186.5 36.963v-20.22h-5.8v-4.64h16.573v4.64h-5.801v20.22H186.5ZM162.725 36.93V12.07h14.253v4.64h-9.281v5.171h7.856v4.641h-7.856v5.768h9.347v4.64h-14.319ZM140.813 36.93V12.07h4.972v10.077h7.193V12.07h4.972v24.86h-4.972V26.787h-7.193V36.93h-4.972ZM127.675 37.262c-5.039 0-8.585-3.182-8.585-8.52v-8.617c0-5.27 3.58-8.387 8.585-8.387 5.005 0 8.585 3.116 8.585 8.387v1.657h-4.972v-1.525c0-2.353-1.459-3.878-3.613-3.878-2.155 0-3.613 1.525-3.613 3.878v8.486c0 2.353 1.458 3.878 3.613 3.878 2.154 0 3.613-1.525 3.613-3.878V26.82h4.972v1.923c0 5.337-3.547 8.519-8.585 8.519ZM105.132 36.963v-20.22h-5.8v-4.64h16.573v4.64h-5.801v20.22h-4.972ZM79.94 36.93l6.165-24.86h6.397l6.166 24.86H92.8l-.663-3.348h-6.033l-.663 3.348H79.94Zm6.96-7.325h4.475l-2.22-11.27-2.254 11.27ZM59.088 36.93V12.07h4.973v10.077h7.192V12.07h4.972v24.86h-4.972V26.787h-7.192V36.93h-4.973Z"
      />
    </svg>
  ),
  head: () => {
    const { title } = useConfig();
    const router = useRouter();

    const fallbackTitle = "Hatchet Documentation";

    // Build the path to the LLM-friendly markdown version of this page
    const pathname = router.pathname.replace(/^\//, "").replace(/\/$/, "") || "index";
    const llmsMarkdownHref = `/llms/${pathname}.md`;

    return (
      <>
        <title>{title ? `${title} - ${fallbackTitle}` : fallbackTitle}</title>
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <link rel="icon" type="image/png" href="/favicon.ico" />
        <link rel="alternate" type="text/markdown" href={llmsMarkdownHref} />
      </>
    );
  },
  main: ({ children }) => {
    const router = useRouter();
    const { setTheme } = useTheme();
    const [origin, setOrigin] = useState(() =>
      typeof window !== "undefined" ? window.location.origin : DEFAULT_ORIGIN
    );

    useEffect(() => {
      const themeParam = router.query.theme;

      if (themeParam === "dark" || themeParam === "light") {
        setTheme(themeParam);
      }
    }, [router.query.theme, setTheme]);

    const pathname =
      router.pathname.replace(/^\//, "").replace(/\/$/, "") || "index";
    const llmsMarkdownHref = `/llms/${pathname}.md`;

    const mcpUrl = `${origin}/api/mcp`;
    const cursorConfig = JSON.stringify({
      command: "npx",
      args: ["-y", "mcp-remote", mcpUrl],
    });
    const cursorDeeplink = `cursor://anysphere.cursor-deeplink/mcp/install?name=hatchet-docs&config=${safeBase64Encode(cursorConfig)}`;

    const claudeCommand = `claude mcp add --transport http hatchet-docs ${mcpUrl}`;

    return (
      <div style={{ position: "relative" }}>
        <div className="page-actions">
          <a href={cursorDeeplink} style={pageLinkStyle} onClick={() => posthog.capture("mcp_install_click", { editor: "cursor", method: "deeplink", page: pathname })} title="Add to Cursor">
            <CursorIcon />
            <span className="page-action-label">Add to Cursor</span>
          </a>
          <CopyClaudeButton command={claudeCommand} />
          <a href={llmsMarkdownHref} target="_blank" rel="noopener noreferrer" style={pageLinkStyle} onClick={() => posthog.capture("docs_view_markdown", { page: pathname })} title="View as Markdown">
            <MarkdownIcon />
            <span className="page-action-label">Raw</span>
          </a>
        </div>
        {children}
      </div>
    );
  },
  primaryHue: {
    dark: 210,
    light: 210,
  },
  primarySaturation: {
    dark: 60,
    light: 60,
  },
  logoLink: "https://hatchet.run",
  project: {
    link: "https://github.com/hatchet-dev/hatchet",
  },
  chat: {
    link: "https://hatchet.run/discord",
  },
  docsRepositoryBase:
    "https://github.com/hatchet-dev/hatchet/blob/main/frontend/docs",
  feedback: {
    labels: "Feedback",
    useLink: (...args: unknown[]) =>
      `https://github.com/hatchet-dev/hatchet/issues/new`,
  },
  footer: false,
  sidebar: {
    defaultMenuCollapseLevel: 2,
    toggleButton: true,
  },
  search: {
    component: Search,
  },
  darkMode: true,
  nextThemes: {
    defaultTheme: "dark",
  },
  themeSwitch: {
    useOptions() {
      return {
        dark: "Dark",
        light: "Light",
      };
    },
  },
};

export default config;
