import type { ComponentPropsWithRef } from "react";

import { Coins } from "lucide-react";

import type { BadgeProps } from "./Badge";

import { Badge } from "./Badge";

export type TokensBadgeProps = ComponentPropsWithRef<"span"> & {
  tokensCount: number;
  size?: BadgeProps["size"];
};

export const TokensBadge = ({
  tokensCount,
  size,
  ...rest
}: TokensBadgeProps) => {
  return (
    <Badge
      iconStart={<Coins className="size-2.5" />}
      size={size}
      {...rest}
      label={tokensCount}
    />
  );
};
