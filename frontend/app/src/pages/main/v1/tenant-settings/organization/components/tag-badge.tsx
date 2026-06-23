import { Badge } from '@/components/v1/ui/badge';

export function TagBadge({ tag }: { tag: string }) {
  const colonIdx = tag.indexOf(':');
  if (colonIdx === -1) {
    return (
      <Badge variant="secondary" className="text-xs">
        {tag}
      </Badge>
    );
  }
  const key = tag.slice(0, colonIdx);
  const value = tag.slice(colonIdx + 1);
  return (
    <Badge variant="secondary" className="text-xs font-normal">
      <span className="text-muted-foreground">{key}</span>
      <span className="mx-px text-muted-foreground">:</span>
      <span className="font-medium">{value}</span>
    </Badge>
  );
}
