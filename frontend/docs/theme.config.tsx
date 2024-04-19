import React from "react";
import { DocsThemeConfig } from "nextra-theme-docs";
import Image from "next/image";

const config: DocsThemeConfig = {
  logo: (
    <Image src="/hatchet_logo.png" alt="Hatchet logo" width={120} height={35} />
  ),
  primaryHue: 210,
  primarySaturation: 60,
  logoLink: "https://github.com/hatchet-dev/hatchet",
  project: {
    link: "https://github.com/hatchet-dev/hatchet",
  },
  chat: {
    link: "https://discord.gg/ZMeUafwH89",
  },
  docsRepositoryBase: "https://github.com/hatchet-dev/hatchet/frontend/docs",
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
      <title>Hatchet Documentation</title>{" "}
    </>
  ),
  sidebar: {
    defaultMenuCollapseLevel: 2,
  },
};

export default config;
