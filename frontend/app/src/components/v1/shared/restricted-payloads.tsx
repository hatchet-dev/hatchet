export function RestrictedPayloads({
  description = 'Your role on this tenant does not include payload access.',
}: {
  description?: string;
}) {
  return (
    <div className="rounded-md border border-dashed p-4 text-sm text-muted-foreground">
      <p className="font-medium text-foreground">Payloads hidden</p>
      <p className="mt-1">{description}</p>
    </div>
  );
}

export function isPayloadsRestricted(value?: {
  payloadsRestricted?: boolean;
}): boolean {
  return value?.payloadsRestricted === true;
}
