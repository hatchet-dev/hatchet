import React, { useEffect } from "react";
import Image from "next/image";
import { useConfig, useTheme } from "nextra-theme-docs";
import { useRouter } from "next/router";

const config = {
  logo: (
    <Image src="/hatchet_logo.png" alt="Hatchet logo" width={120} height={35} />
  ),
  head: () => {
    const { title } = useConfig();

    const fallbackTitle = "Hatchet Documentation";

    return (
      <>
        <title>{title ? `${title} - ${fallbackTitle}` : fallbackTitle}</title>
        <meta name="viewport" content="width=device-width, initial-scale=1.0" />
        <link rel="icon" type="image/png" href="/favicon.ico" />
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

    return <>{children}</>;
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
