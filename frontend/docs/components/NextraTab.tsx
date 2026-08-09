"use client";

import React from "react";
import { Tab as FumaTab } from "fumadocs-ui/components/tabs";

export function NextraTab({
  title,
  value,
  label,
  children,
  ...props
}: {
  title?: string;
  value?: string;
  label?: string;
  children?: React.ReactNode;
} & Omit<React.ComponentProps<typeof FumaTab>, "title" | "value">) {
  return <FumaTab {...props}>{children}</FumaTab>;
}
