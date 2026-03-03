import { Check, Copy, X } from "lucide-react";
import { useState } from "react";

import { IconButton } from "./IconButton";

type CopyButtonProps = {
  label: string;
  content: string;
};

type CopyState = "idle" | "success" | "error";

export const CopyButton = ({ label, content }: CopyButtonProps) => {
  const [copyState, setCopyState] = useState<CopyState>("idle");

  const onClick = async () => {
    try {
      if (!navigator.clipboard) {
        throw new Error("Clipboard API not supported");
      }

      await navigator.clipboard.writeText(content);
      setCopyState("success");
      setTimeout(() => setCopyState("idle"), 2000);
    } catch {
      setCopyState("error");
      setTimeout(() => setCopyState("idle"), 2000);
    }
  };

  const getIcon = () => {
    switch (copyState) {
      case "success":
        return <Check className="size-3" />;
      case "error":
        return <X className="size-3" />;
      default:
        return <Copy className="size-3" />;
    }
  };

  const getAriaLabel = () => {
    switch (copyState) {
      case "success":
        return `${label} Copied`;
      case "error":
        return `Failed to copy ${label}`;
      default:
        return `Copy ${label}`;
    }
  };

  return (
    <IconButton
      onClick={onClick}
      aria-label={getAriaLabel()}
      variant="ghost"
      disabled={copyState !== "idle"}
    >
      {getIcon()}
    </IconButton>
  );
};
