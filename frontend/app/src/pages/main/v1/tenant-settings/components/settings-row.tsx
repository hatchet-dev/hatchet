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
    <div className="flex flex-wrap items-start justify-between gap-x-8 gap-y-3 py-4">
      <div className="min-w-[16rem] flex-1">
        <p className="text-sm font-medium">{label}</p>
        {description && (
          <p className="mt-0.5 max-w-xl text-xs text-muted-foreground">
            {description}
          </p>
        )}
      </div>
      <div className="ml-auto flex shrink-0 flex-wrap justify-end gap-2">
        {children}
      </div>
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
