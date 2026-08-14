"use client";

import { useEffect } from "react";
import { useSearchParams } from "next/navigation";
import { useTheme } from "next-themes";

export function ThemeQueryParam() {
  const searchParams = useSearchParams();
  const { setTheme } = useTheme();

  useEffect(() => {
    const themeParam = searchParams.get("theme");
    if (
      themeParam === "dark" ||
      themeParam === "light" ||
      themeParam === "system"
    ) {
      setTheme(themeParam);
    }
  }, [searchParams, setTheme]);

  return null;
}
