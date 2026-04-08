import React from "react";

export function SsoErrorText({ children }: { children?: React.ReactNode }) {
    if (!children) return null;
    return <p className="text-xs text-destructive">{children}</p>;
}
