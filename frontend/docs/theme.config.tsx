import React from "react";
import { DocsThemeConfig } from "nextra-theme-docs";
import Image from "next/image";

const config: DocsThemeConfig = {
  logo: (
    <Image src="/hatchet_logo.png" alt="Hatchet logo" width={120} height={35} />
  ),
  logoLink: "https://hatchet.run",
  primaryHue: 210,
  primarySaturation: 10,
  project: {
    link: "https://github.com/hatchet-dev/hatchet",
  },
  chat: {
    link: "https://discord.gg/ZMeUafwH89",
  },
  docsRepositoryBase: "https://github.com/hatchet-dev/hatchet/frontend/docs",
  footer: {
    text: "Hatchet",
  },
  head: (
    <>
      <link rel="icon" type="image/png" href="/favicon.ico" />
      <title>Hatchet Documentation</title>{" "}
    </>
  ),
  sidebar: {
    defaultMenuCollapseLevel: 1,
  },
  nextThemes: {
    forcedTheme: "dark",
  },
  themeSwitch: {
    component: () => null,
  },
};

export default config;
