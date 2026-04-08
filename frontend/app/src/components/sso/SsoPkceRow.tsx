import { Switch } from "@/components/ui/switch";

export function SsoPkceRow({ checked, onChange }: { checked: boolean; onChange: (v: boolean) => void }) {
    return (
        <div className="mt-2 flex items-center justify-between rounded-xl border p-3">
            <div className="space-y-0.5">
                <div className="text-sm font-medium">Use PKCE</div>
                <p className="text-xs text-muted-foreground">Recommended.</p>
            </div>
            <Switch checked={checked} onCheckedChange={onChange} />
        </div>
    );
}
