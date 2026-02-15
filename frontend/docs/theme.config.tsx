import React, { useEffect } from "react";
import { useConfig, useTheme } from "nextra-theme-docs";
import { useRouter } from "next/router";

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

    useEffect(() => {
      const themeParam = router.query.theme;

      if (themeParam === "dark" || themeParam === "light") {
        setTheme(themeParam);
      }
    }, [router.query.theme, setTheme]);

    const pathname =
      router.pathname.replace(/^\//, "").replace(/\/$/, "") || "index";
    const llmsMarkdownHref = `/llms/${pathname}.md`;

    return (
      <>
        <div style={{ display: "flex", justifyContent: "flex-end", marginBottom: "0.5rem" }}>
          <a
            href={llmsMarkdownHref}
            target="_blank"
            rel="noopener noreferrer"
            style={{
              fontSize: "0.75rem",
              opacity: 0.5,
              textDecoration: "none",
            }}
          >
            View as Markdown
          </a>
        </div>
        {children}
      </>
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
