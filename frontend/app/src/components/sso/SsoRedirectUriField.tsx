import { useState } from "react";
import { Check, Copy } from "lucide-react";
import { Button } from "@/components/v1/ui/button";
import { Input } from "@/components/v1/ui/input";
import { SsoField } from "./SsoField";
import { copySsoToClipboard } from "@/lib/sso/sso-utils";

export function SsoRedirectUriField({ redirectUrl }: { redirectUrl: string }) {
    const [copied, setCopied] = useState(false);

    return (
        <SsoField label="Redirect / Callback URL">
            <div className="flex items-center gap-2">
                <Input readOnly value={redirectUrl} tabIndex={-1} />
                <Button
                    type="button"
                    size="sm"
                    onClick={() => {
                        copySsoToClipboard(redirectUrl, () => {
                            setCopied(true);
                            setTimeout(() => setCopied(false), 500);
                        });
                    }}
                    className="shrink-0 cursor-pointer"
                >
                    {copied ? <Check className="h-4 w-4" /> : <Copy className="h-4 w-4" />}
                </Button>
            </div>
        </SsoField>
    );
}
