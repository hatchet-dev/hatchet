"use client";

import React from "react";
import { Tab as FumaTab } from "fumadocs-ui/components/tabs";

/**
 * Client wrapper for fumadocs' Tab that accepts Nextra's `title` prop.
 * Must be a client component: UniversalTabs introspects `child.props.title`
 * on its children, which only survives for client-component elements.
 */
export function NextraTab({
  title,
  value,
  label,
  children,
  ...props
}: {
  title?: string;
  /* value/label were silently ignored by the old Tabs; keep ignoring them
     so fumadocs' index-based tab pairing applies */
  value?: string;
  label?: string;
  children?: React.ReactNode;
} & Omit<React.ComponentProps<typeof FumaTab>, "title" | "value">) {
  return <FumaTab {...props}>{children}</FumaTab>;
}
