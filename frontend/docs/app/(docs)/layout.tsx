import { DocsLayout } from "fumadocs-ui/layouts/docs";
import { getLayoutTabs } from "fumadocs-ui/layouts/shared";
import { Cloud } from "lucide-react";
import type { ReactNode } from "react";
import { source } from "@/lib/source";
import { HatchetLogo } from "@/components/HatchetLogo";
import { SidebarLanguageSelect } from "@/components/SidebarLanguageSelect";

const HIDDEN_TABS = new Set(["Contributing", "Agent Instructions"]);

function wrapIcon(icon: ReactNode) {
  if (!icon) return undefined;
  return (
    <div className="size-full [&_svg]:size-full max-md:p-1.5 max-md:rounded-md max-md:border max-md:bg-fd-secondary">
      {icon}
    </div>
  );
}

export default function Layout({ children }: { children: ReactNode }) {
  const tree = source.getPageTree();

  const tabs = getLayoutTabs(tree, {
    transform: (option, node) => {
      if (HIDDEN_TABS.has(String(option.title))) return null;
      return { ...option, icon: wrapIcon(node.icon) ?? option.icon };
    },
  });

  // The Reference folder has no index page (so it gets no sidebar entry of its
  // own) which also means getLayoutTabs skips it, so its tab is added manually,
  // pointing at the changelog. $folder keeps active-state detection working
  // for every /reference/* page.
  const reference = tree.children.find(
    (node) => node.type === "folder" && String(node.name) === "Reference",
  );
  if (reference && reference.type === "folder") {
    tabs.push({
      title: reference.name,
      description: reference.description,
      url: "/reference/changelog",
      icon: wrapIcon(reference.icon),
      $folder: reference,
    });
  }

  return (
    <DocsLayout
      tree={tree}
      tabs={tabs}
      nav={{ title: <HatchetLogo />, url: "https://hatchet.run" }}
      sidebar={{
        defaultOpenLevel: 0,
        banner: <SidebarLanguageSelect />,
        footer: (
          <a
            href="https://cloud.hatchet.run?utm_source=docs&utm_medium=sidebar"
            target="_blank"
            rel="noreferrer noopener"
            className="flex grow items-center gap-2 rounded-md px-2 py-1.5 text-[0.8125rem] text-fd-muted-foreground transition-colors hover:text-fd-accent-foreground"
          >
            <Cloud className="size-4" />
            Hatchet Cloud
          </a>
        ),
      }}
    >
      {children}
    </DocsLayout>
  );
}
