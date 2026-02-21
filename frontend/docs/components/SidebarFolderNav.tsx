"use client";

import { useRouter } from "next/router";
import { useEffect } from "react";

/**
 * Nextra renders sidebar folders with index pages as buttons. When a collapsed
 * folder is clicked, we navigate to its index (data-href) so the user lands
 * on the overview; the sidebar then opens because the route is inside the folder.
 * Works for any folder that has a route (e.g. has an index.mdx).
 */
export function SidebarFolderNav() {
  const router = useRouter();

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      const target = e.target as HTMLElement;
      const button = target.closest?.(
        ".nextra-sidebar-container button[data-href]"
      ) as HTMLButtonElement | null;
      if (!button) return;

      const href = button.getAttribute("data-href");
      if (!href) return;

      const li = button.closest?.("li");
      const isOpen = li?.classList.contains("open");
      if (isOpen) return;

      e.preventDefault();
      e.stopPropagation();
      router.push(href);
    }

    document.addEventListener("click", handleClick, true);
    return () => document.removeEventListener("click", handleClick, true);
  }, [router]);

  return null;
}
