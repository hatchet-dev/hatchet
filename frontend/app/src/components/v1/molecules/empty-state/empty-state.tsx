import { DocPage } from '@/components/v1/docs/docs-button';
import { ExternalLink } from 'lucide-react';

type EmptyStateActionBase = {
  icon?: React.ReactNode;
  label: string;
  description?: string;
};

type EmptyStateAction = EmptyStateActionBase &
  (
    | { href: string; external?: boolean; onClick?: never }
    | { onClick: () => void; href?: never; external?: never }
  );

type InlineLink = {
  href: string;
  label: string;
  external?: boolean;
};

type EmptyStateProps = {
  title: string;
  description: string;
  docPage?: DocPage;
  docLabel?: string;
  links?: InlineLink[];
  graphic?: React.ReactNode;
  graphicPosition?: 'top' | 'bottom';
  filterHint?: string;
  actions?: EmptyStateAction[];
};

const actionCardClass =
  'flex items-start gap-3 rounded-lg border border-border bg-background p-4 hover:bg-muted/50 transition-colors text-left no-underline cursor-pointer flex-1 basis-44';

function ActionCardInner({ action }: { action: EmptyStateAction }) {
  return (
    <>
      {action.icon && (
        <span className="mt-0.5 flex-shrink-0 text-muted-foreground">
          {action.icon}
        </span>
      )}
      <span className="flex min-w-0 flex-col gap-0.5">
        <span className="text-sm font-medium text-foreground">{action.label}</span>
        {action.description && (
          <span className="text-xs text-muted-foreground">{action.description}</span>
        )}
      </span>
      {'external' in action && action.external && (
        <ExternalLink className="ml-auto size-3 flex-shrink-0 text-muted-foreground" />
      )}
    </>
  );
}

function ActionCard({ action }: { action: EmptyStateAction }) {
  if ('onClick' in action && action.onClick) {
    return (
      <button type="button" onClick={action.onClick} className={actionCardClass}>
        <ActionCardInner action={action} />
      </button>
    );
  }

  if (action.external) {
    return (
      <a href={action.href} target="_blank" rel="noreferrer" className={actionCardClass}>
        <ActionCardInner action={action} />
      </a>
    );
  }

  return (
    <a href={action.href} className={actionCardClass}>
      <ActionCardInner action={action} />
    </a>
  );
}

export function EmptyState({
  title,
  description,
  docPage,
  docLabel,
  links,
  graphic,
  graphicPosition = 'top',
  filterHint,
  actions,
}: EmptyStateProps) {
  const graphicNode = graphic && <div>{graphic}</div>;

  return (
    <div className="flex flex-col items-center justify-center gap-y-3 text-foreground">
      {graphicPosition === 'top' && graphicNode}
      <p className="text-lg font-semibold">{title}</p>
      {filterHint && (
        <p className="text-xs italic text-muted-foreground/70">{filterHint}</p>
      )}
      <p className="max-w-sm text-center text-sm text-muted-foreground">{description}</p>
      {!actions && links && links.length > 0 && (
        <div className="flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
          {links.map((link, i) => (
            <a
              key={i}
              href={link.href}
              target={link.external ? '_blank' : undefined}
              rel={link.external ? 'noreferrer' : undefined}
              className="inline-flex items-center gap-1 text-sm text-blue-500 hover:underline"
            >
              {link.label}
              {link.external && <ExternalLink className="size-3" />}
            </a>
          ))}
        </div>
      )}
      {!actions && !links && docPage && docLabel && (
        <a
          href={docPage.href}
          target="_blank"
          rel="noreferrer"
          className="inline-flex items-center gap-1 text-sm text-blue-500 hover:underline"
        >
          {docLabel}
          <ExternalLink className="size-3" />
        </a>
      )}
      {actions && actions.length > 0 && (
        <div className="mt-1 flex w-full max-w-lg flex-row flex-wrap gap-3">
          {actions.map((action, i) => (
            <ActionCard key={i} action={action} />
          ))}
        </div>
      )}
      {graphicPosition === 'bottom' && graphicNode}
    </div>
  );
}
