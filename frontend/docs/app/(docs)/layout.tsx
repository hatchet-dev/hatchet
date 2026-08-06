import { DocsLayout } from "fumadocs-ui/layouts/docs";
import type { ReactNode } from "react";
import { source } from "@/lib/source";
import { HatchetLogo } from "@/components/HatchetLogo";

const HIDDEN_TABS = new Set(["Contributing", "Agent Instructions"]);

export default function Layout({ children }: { children: ReactNode }) {
  return (
    <DocsLayout
      tree={source.getPageTree()}
      tabs={{
        transform: (option, node) => {
          if (HIDDEN_TABS.has(String(option.title))) return null;
          const icon = node.icon ? (
            <div className="size-full [&_svg]:size-full max-md:p-1.5 max-md:rounded-md max-md:border max-md:bg-fd-secondary">
              {node.icon}
            </div>
          ) : (
            option.icon
          );
          // the reference folder has no index page; without this the tab
          // falls through to its first link entry (the external Go SDK)
          if (option.title === "Reference") {
            return { ...option, icon, url: "/reference/changelog" };
          }
          return { ...option, icon };
        },
      }}
      nav={{ title: <HatchetLogo />, url: "https://hatchet.run" }}
      githubUrl="https://github.com/hatchet-dev/hatchet"
      sidebar={{ defaultOpenLevel: 0 }}
    >
      {children}
    </DocsLayout>
  );
}
