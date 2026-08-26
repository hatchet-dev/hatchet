import React from "react";
import styles from "./text.module.css";

/**
 * Minimal compat shim for the marketing site's `Text` component, covering
 * only the surface the ported Flow diagrams use (`Text.Small` / `Text.Micro`
 * captions and labels). Renders marketing typography off the `flow-scope`
 * tokens, so it must sit inside a Flow frame (or another `.flow-scope`).
 */

interface TextProps extends React.HTMLAttributes<HTMLElement> {
  className?: string;
  children: React.ReactNode;
  as?: React.ElementType;
  balance?: boolean;
  brackets?: boolean;
  secondary?: boolean;
  tertiary?: boolean;
  primary?: boolean;
  mono?: boolean;
  caps?: boolean;
}

const createStyledText = (variant: "small" | "micro") => {
  const Component = ({
    children,
    className,
    as: Tag = "span",
    balance,
    brackets,
    secondary,
    tertiary,
    primary,
    mono,
    caps,
    ...props
  }: TextProps) => {
    const finalClassName = [
      styles[variant],
      className || "",
      balance ? styles.balance : "",
      secondary ? styles.secondary : "",
      tertiary ? styles.tertiary : "",
      primary ? styles.primary : "",
      brackets ? styles.brackets : "",
      mono ? styles.mono : "",
      caps ? styles.caps : "",
    ]
      .filter(Boolean)
      .join(" ");
    return (
      <Tag className={finalClassName} {...props}>
        {children}
      </Tag>
    );
  };
  Component.displayName = `Text.${variant}`;
  return Component;
};

export const Text = {
  Small: createStyledText("small"),
  Micro: createStyledText("micro"),
};
