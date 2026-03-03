import type { ComponentPropsWithRef } from "react";

import type { BadgeProps } from "./Badge";

import { Badge } from "./Badge";

export type PriceBadgeProps = ComponentPropsWithRef<"span"> & {
  cost: number;
  size?: BadgeProps["size"];
};

export const PriceBadge = ({ cost, size, ...rest }: PriceBadgeProps) => {
  return <Badge size={size} {...rest} label={`$ ${cost}`} />;
};
