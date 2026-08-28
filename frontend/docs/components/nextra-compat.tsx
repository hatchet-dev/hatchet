import React from "react";
import {
  Callout as FumaCallout,
  type CalloutType,
} from "fumadocs-ui/components/callout";
import {
  Cards as FumaCards,
  Card as FumaCard,
} from "fumadocs-ui/components/card";
import {
  Files as FumaFiles,
  Folder as FumaFolder,
  File as FumaFile,
} from "fumadocs-ui/components/files";
import { CompatTabs } from "./CompatTabs";
import { NextraTab } from "./NextraTab";

export function Tabs(props: React.ComponentProps<typeof CompatTabs>) {
  return <CompatTabs {...props} />;
}
Tabs.Tab = NextraTab;

const CALLOUT_TYPE_MAP: Record<string, CalloutType> = {
  default: "info",
  info: "info",
  warning: "warn",
  error: "error",
};

export function Callout({
  type = "default",
  emoji,
  children,
  ...props
}: {
  type?: string;
  emoji?: React.ReactNode;
  children?: React.ReactNode;
} & Omit<React.ComponentProps<typeof FumaCallout>, "type" | "icon">) {
  return (
    <FumaCallout
      type={CALLOUT_TYPE_MAP[type] ?? "info"}
      icon={emoji ? <span>{emoji}</span> : undefined}
      {...props}
    >
      {children}
    </FumaCallout>
  );
}

export function Steps({ children }: { children?: React.ReactNode }) {
  return <div className="fd-steps [&_h3]:fd-step">{children}</div>;
}

function CardsCard({
  title,
  href,
  icon,
  children,
  ...props
}: {
  title?: React.ReactNode;
  href?: string;
  icon?: React.ReactNode;
  arrow?: boolean;
  children?: React.ReactNode;
}) {
  return (
    <FumaCard title={title} href={href} icon={icon} {...props}>
      {children}
    </FumaCard>
  );
}

export function Cards({
  children,
  ...props
}: React.ComponentProps<typeof FumaCards>) {
  return <FumaCards {...props}>{children}</FumaCards>;
}
Cards.Card = CardsCard;

export const Card = CardsCard;

export function Code(props: React.ComponentProps<"code">) {
  return <code {...props} />;
}

export function Bleed({
  full,
  children,
}: {
  full?: boolean;
  children?: React.ReactNode;
}) {
  return (
    <div className={full ? "w-full" : "mx-[-1rem] md:mx-[-2rem]"}>
      {children}
    </div>
  );
}

export function FileTree({ children }: { children?: React.ReactNode }) {
  return <FumaFiles>{children}</FumaFiles>;
}
FileTree.Folder = function Folder({
  name,
  defaultOpen,
  children,
}: {
  name: string;
  defaultOpen?: boolean;
  children?: React.ReactNode;
}) {
  return (
    <FumaFolder name={name} defaultOpen={defaultOpen}>
      {children}
    </FumaFolder>
  );
};
FileTree.File = function File({ name }: { name: string }) {
  return <FumaFile name={name} />;
};
