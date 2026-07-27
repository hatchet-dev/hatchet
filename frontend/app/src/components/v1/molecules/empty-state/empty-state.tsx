import { DocPage } from '@/components/v1/docs/docs-button';
import { Button } from '@/components/v1/ui/button';
import { ExternalLink } from 'lucide-react';

type EmptyStateActionBase = {
  icon?: React.ReactNode;
  label: string;
  description?: string;
};

export type EmptyStateAction = EmptyStateActionBase &
  (
    | { href: string; external?: boolean; onClick?: never }
    | { onClick: () => void; href?: never; external?: never }
  );

type InlineLink = {
  href: string;
  label: string;
  external?: boolean;
};

type EmptyStateButton = { label: string; onClick: () => void };

type EmptyStateProps = {
  title: string;
  description?: string;
  docPage?: DocPage;
  docLabel?: string;
  links?: InlineLink[];
  graphic?: React.ReactNode;
  graphicPosition?: 'top' | 'bottom';
  filterHint?: string;
  actions?: EmptyStateAction[];
  buttons?: EmptyStateButton[];
};

const actionCardClass =
  'flex min-w-0 flex-1 basis-44 cursor-pointer flex-col items-start gap-1.5 rounded-lg border border-[#053970]/[0.16] bg-[#005c9e]/[0.02] px-4 py-3 text-left no-underline transition-all hover:border-[#004c8c]/10 hover:bg-white hover:shadow-[0_3px_2px_rgba(0,0,0,0.01),0_8px_11px_rgba(0,0,0,0.03),0_24px_16px_rgba(0,0,0,0.01)] dark:border-[#97c4ff]/[0.16] dark:bg-[#82b9ff]/[0.07] dark:hover:border-[#86bbff]/[0.08] dark:hover:bg-[#8dbfff]/[0.13]';

function ActionCardInner({ action }: { action: EmptyStateAction }) {
  return (
    <>
      <span className="flex w-full items-center gap-2">
        {action.icon && (
          <span className="flex-shrink-0 text-muted-foreground">
            {action.icon}
          </span>
        )}
        <span className="truncate text-xs font-medium text-foreground">
          {action.label}
        </span>
        {'external' in action && action.external && (
          <ExternalLink className="ml-auto size-3 flex-shrink-0 text-muted-foreground" />
        )}
      </span>
      {action.description && (
        <span className="text-xs leading-normal text-muted-foreground">
          {action.description}
        </span>
      )}
    </>
  );
}

function ActionCard({ action }: { action: EmptyStateAction }) {
  if ('onClick' in action && action.onClick) {
    return (
      <button
        type="button"
        onClick={action.onClick}
        className={actionCardClass}
      >
        <ActionCardInner action={action} />
      </button>
    );
  }

  if (action.external) {
    return (
      <a
        href={action.href}
        target="_blank"
        rel="noreferrer"
        className={actionCardClass}
      >
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
  buttons,
}: EmptyStateProps) {
  const graphicNode = graphic && <div>{graphic}</div>;

  return (
    <div className="flex flex-col items-center justify-center gap-y-3 text-foreground">
      {graphicPosition === 'top' && graphicNode}
      <p className="text-lg font-semibold">{title}</p>
      {filterHint && (
        <p className="text-xs italic text-muted-foreground/70">{filterHint}</p>
      )}
      {description && (
        <p className="max-w-sm text-center text-sm text-muted-foreground">
          {description}
        </p>
      )}
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
      {buttons && buttons.length > 0 && (
        <div className="flex flex-wrap items-center justify-center gap-2">
          {buttons.map((btn, i) => (
            <Button key={i} variant="outline" size="sm" onClick={btn.onClick}>
              {btn.label}
            </Button>
          ))}
        </div>
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
