import { Label } from "@/components/ui/label";

export function SsoField({
    label,
    htmlFor,
    children,
    required,
}: {
    label: string;
    htmlFor?: string;
    children: React.ReactNode;
    required?: boolean;
}) {
    return (
        <div className="grid gap-1.5">
            <Label htmlFor={htmlFor}>
                {label} {required && <span className="text-destructive">*</span>}
            </Label>
            {children}
        </div>
    );
}
