import CopyToClipboard from '@/components/v1/ui/copy-to-clipboard';

export function SettingRow({
  label,
  description,
  children,
}: {
  label: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between gap-4 py-4">
      <div>
        <p className="text-sm font-medium">{label}</p>
        {description && (
          <p className="text-xs text-muted-foreground mt-0.5">{description}</p>
        )}
      </div>
      {children}
    </div>
  );
}

export function ReadOnlyValue({ value }: { value: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="max-w-[280px] truncate font-mono text-sm text-muted-foreground">
        {value}
      </span>
      <CopyToClipboard text={value} />
    </div>
  );
}
