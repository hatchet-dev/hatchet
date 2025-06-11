const BASE_DOCS_URL = 'https://docs.hatchet.run';

function createDocsExternalLink(path: string) {
  return `${BASE_DOCS_URL}${path}`;
}

type DocExternalLinkProps = {
  link: string;
  description: string;
  className?: string;
};

const DocExternalLink: React.FC<DocExternalLinkProps> = ({
  link,
  description,
  className,
}) => {
  return (
    <a
      target="_blank"
      rel="noopener noreferrer"
      href={createDocsExternalLink(link)}
      className={className}
    >
      <span className="text-indigo-500">{description}</span>
    </a>
  );
};

type IntroDocsEmptyStateProps = {
  link: string;
  linkText: string;
  linkPreambleText: string;
  title?: string;
};

export const IntroDocsEmptyState: React.FC<IntroDocsEmptyStateProps> = ({
  link,
  linkText,
  linkPreambleText,
  title,
}) => {
  return (
    <div className="w-full h-full flex flex-col gap-y-4 text-foreground py-8">
      {title && <p className="text-lg font-bold">{title}</p>}
      <p className="text-sm">
        {linkPreambleText}{' '}
        <DocExternalLink
          link={link}
          description={linkText}
          className="inline"
        />
      </p>
    </div>
  );
};
