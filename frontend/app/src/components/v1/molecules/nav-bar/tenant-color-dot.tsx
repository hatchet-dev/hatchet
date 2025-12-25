const DEFAULT_TENANT_COLOR = '#3B82F6';

export function TenantColorDot({
  color,
  size = 'md',
}: {
  color?: string;
  size?: 'sm' | 'md';
}) {
  const sizeClass = size === 'sm' ? 'size-2' : 'size-3';

  return (
    <span
      aria-hidden
      className={`${sizeClass} shrink-0 rounded-full`}
      style={{ backgroundColor: color || DEFAULT_TENANT_COLOR }}
    />
  );
}

export { DEFAULT_TENANT_COLOR };
