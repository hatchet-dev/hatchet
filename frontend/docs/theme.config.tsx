import React from "react";
import { DocsThemeConfig } from "nextra-theme-docs";
import Image from "next/image";
import FrameworkSelector from "./components/FrameworkSelector";

const config: DocsThemeConfig = {
  logo: (
    <Image src="/hatchet_logo.png" alt="Hatchet logo" width={120} height={35} />
  ),
  useNextSeoProps() {
    return {
      titleTemplate: '%s – Hatchet Docs'
    }
  },
  primaryHue: 210,
  primarySaturation: 60,
  logoLink: "https://github.com/hatchet-dev/hatchet",
  project: {
    link: "https://github.com/hatchet-dev/hatchet",
  },
  chat: {
    link: "https://discord.gg/ZMeUafwH89",
  },
  docsRepositoryBase: "https://github.com/hatchet-dev/hatchet/blob/main/frontend/docs",
  feedback: {
    labels: "Feedback",
    useLink: (...args: unknown[]) =>
      `https://github.com/hatchet-dev/hatchet/issues/new`,
  },
  footer: {
    text: "Hatchet",
  },
  head: (
    <>
      <link rel="icon" type="image/png" href="/favicon.ico" />
    </>
  ),
  sidebar: {
    defaultMenuCollapseLevel: 2,
    titleComponent: ({ title, type, route }) => {
      if (type === 'separator' && title === 'Frameworks') {
        return <FrameworkSelector />
      }
      return <div>{title}</div>
    }
  },
};

export default config;
