"use client";

import { useRouter } from "next/router";
import { useEffect } from "react";

/**
 * Nextra renders sidebar folders with index pages as buttons. When the folder
 * label is clicked (expanded or collapsed), we navigate to its index (data-href)
 * so the user lands on the overview.
 * Works for any folder that has a route (e.g. has an index.mdx).
 */
export function SidebarFolderNav() {
  const router = useRouter();

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      const target = e.target as HTMLElement;
      const button = target.closest?.(
        ".nextra-sidebar-container button[data-href]",
      ) as HTMLButtonElement | null;
      if (!button) return;

      const href = button.getAttribute("data-href");
      if (!href) return;

      e.preventDefault();
      e.stopPropagation();
      router.push(href);
    }

    document.addEventListener("click", handleClick, true);
    return () => document.removeEventListener("click", handleClick, true);
  }, [router]);

  return null;
}
