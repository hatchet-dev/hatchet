import { Button } from '@/components/v1/ui/button';
import useControlPlane from '@/hooks/use-control-plane';
import { getCloudCTAUrl } from '@/lib/external-links';
import { cn } from '@/lib/utils';
import { BiCloud } from 'react-icons/bi';

export function SidebarCloudCTA({
  collapsed,
  className,
}: {
  collapsed: boolean;
  className?: string;
}) {
  const { isSelfHosted, isControlPlaneLoading } = useControlPlane();

  if (isControlPlaneLoading || !isSelfHosted) {
    return null;
  }

  const href = getCloudCTAUrl('sidebar');

  if (collapsed) {
    return (
      <Button
        variant="icon"
        size="icon"
        aria-label="Try Cloud"
        hoverText="Try Cloud"
        hoverTextSide="right"
        className={cn('w-10', className)}
        asChild
      >
        <a href={href} target="_blank" rel="noopener noreferrer">
          <BiCloud className="h-6 w-6 cursor-pointer text-foreground" />
        </a>
      </Button>
    );
  }

  return (
    <Button
      variant="ghost"
      className={cn(
        'w-full justify-start pl-2 min-w-0 overflow-hidden',
        className,
      )}
      asChild
    >
      <a href={href} target="_blank" rel="noopener noreferrer">
        <BiCloud className="size-4 mr-2" />
        <span className="truncate">Try Cloud</span>
      </a>
    </Button>
  );
}
