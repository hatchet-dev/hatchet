import type { TraceSpanCategory } from "@evilmartians/agent-prism-types";
import type { LucideIcon } from "lucide-react";

import {
  Zap,
  Wrench,
  Bot,
  Link,
  Search,
  BarChart2,
  Plus,
  HelpCircle,
  MoveHorizontal,
  CircleDot,
  ShieldCheck,
} from "lucide-react";
import { useEffect, useState } from "react";

// TYPES

export type ColorVariant =
  | "purple"
  | "indigo"
  | "orange"
  | "teal"
  | "cyan"
  | "sky"
  | "yellow"
  | "emerald"
  | "red"
  | "gray";

export type ComponentSize =
  | "4"
  | "5"
  | "6"
  | "7"
  | "8"
  | "9"
  | "10"
  | "11"
  | "12"
  | "16";

// CONSTANTS

export const ROUNDED_CLASSES = {
  none: "rounded-none",
  sm: "rounded-sm",
  md: "rounded-md",
  lg: "rounded-lg",
  full: "rounded-full",
};

/**
 * Shared configuration for span categories containing label, theme, and icon
 */
export const SPAN_CATEGORY_CONFIG: Record<
  TraceSpanCategory,
  {
    label: string;
    theme: ColorVariant;
    icon: LucideIcon;
  }
> = {
  llm_call: {
    label: "LLM",
    theme: "purple",
    icon: Zap,
  },
  tool_execution: {
    label: "TOOL",
    theme: "orange",
    icon: Wrench,
  },
  agent_invocation: {
    label: "AGENT INVOCATION",
    theme: "indigo",
    icon: Bot,
  },
  chain_operation: {
    label: "CHAIN",
    theme: "teal",
    icon: Link,
  },
  retrieval: {
    label: "RETRIEVAL",
    theme: "cyan",
    icon: Search,
  },
  embedding: {
    label: "EMBEDDING",
    theme: "emerald",
    icon: BarChart2,
  },
  create_agent: {
    label: "CREATE AGENT",
    theme: "sky",
    icon: Plus,
  },
  span: {
    label: "SPAN",
    theme: "cyan",
    icon: MoveHorizontal,
  },
  event: {
    label: "EVENT",
    theme: "emerald",
    icon: CircleDot,
  },
  guardrail: {
    label: "GUARDRAIL",
    theme: "red",
    icon: ShieldCheck,
  },
  unknown: {
    label: "UNKNOWN",
    theme: "gray",
    icon: HelpCircle,
  },
};

// UTILS

export function getSpanCategoryTheme(
  category: TraceSpanCategory,
): ColorVariant {
  return SPAN_CATEGORY_CONFIG[category].theme;
}

export function getSpanCategoryLabel(category: TraceSpanCategory): string {
  return SPAN_CATEGORY_CONFIG[category].label;
}

export function getSpanCategoryIcon(category: TraceSpanCategory): LucideIcon {
  return SPAN_CATEGORY_CONFIG[category].icon;
}

export const useIsMobile = () => {
  const isMounted = useIsMounted();

  const [isMobile, setIsMobile] = useState(false);

  useEffect(() => {
    // TODO: replace with something more beautiful and correct (tailwind screens?)
    const mediaQuery = window.matchMedia("(max-width: 1023px)");

    const handleChange = (e: MediaQueryListEvent | MediaQueryList) => {
      setIsMobile(e.matches);
    };

    handleChange(mediaQuery);

    mediaQuery.addEventListener("change", handleChange);

    return () => mediaQuery.removeEventListener("change", handleChange);
  }, []);

  return isMounted ? isMobile : false;
};

export const useIsMounted = () => {
  const [isMounted, setIsMounted] = useState(false);

  useEffect(() => {
    setIsMounted(true);
  }, []);

  return isMounted;
};
