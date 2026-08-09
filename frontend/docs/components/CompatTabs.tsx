"use client";

import React from "react";
import { Tabs as BaseTabs } from "fumadocs-ui/components/ui/tabs";
import {
  TabsList,
  TabsTrigger,
  TabsContent,
} from "fumadocs-ui/components/tabs";

export function CompatTabs({
  items = [],
  defaultIndex = 0,
  children,
}: {
  items?: React.ReactNode[];
  defaultIndex?: number;
  children?: React.ReactNode;
}) {
  const panels = React.Children.toArray(children).filter(
    (child) => !(typeof child === "string" && child.trim() === ""),
  );

  return (
    <BaseTabs
      defaultValue={String(defaultIndex)}
      className="flex flex-col overflow-hidden rounded-xl border bg-fd-secondary my-4"
    >
      <TabsList>
        {items.map((item, i) => (
          <TabsTrigger key={i} value={String(i)}>
            {item}
          </TabsTrigger>
        ))}
      </TabsList>
      {panels.map((panel, i) => (
        <TabsContent key={i} value={String(i)}>
          {React.isValidElement<{ children?: React.ReactNode }>(panel)
            ? panel.props.children
            : panel}
        </TabsContent>
      ))}
    </BaseTabs>
  );
}
