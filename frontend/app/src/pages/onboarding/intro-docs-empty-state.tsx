const BASE_DOCS_URL = 'https://docs.hatchet.run';

const advancedTopicsDocs = [
  {
    link: '/home/features/concurrency/overview',
    description:
      "How to leverage Hatchet's powerful concurrency control features",
  },
  {
    link: '/home/features/timeouts',
    description:
      "Managing timeouts of your tasks to ensure they don't run longer than expected",
  },
  {
    link: '/home/features/durable-execution',
    description:
      "Using Hatchet's durable execution toolkit to ensure your tasks continue despite failures",
  },
];

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
      <span className="text-blue-600">{description}</span>
    </a>
  );
};

type IntroDocsEmptyStateProps = {
  link: string;
  linkText: string;
  linkPreambleText: string;
};

export const IntroDocsEmptyState: React.FC<IntroDocsEmptyStateProps> = ({
  link,
  linkText,
  linkPreambleText,
}) => {
  return (
    <div className="w-full h-full border border-gray rounded-md p-4 flex flex-col justify-between gap-y-12">
      <div className="flex flex-col gap-y-4">
        <p className="text-xl">🪓 Welcome to Hatchet!</p>
        <p>
          {linkPreambleText}{' '}
          <DocExternalLink
            link={link}
            description={linkText}
            className="inline"
          />
        </p>
      </div>
      <div className="flex flex-col gap-y-2">
        <p>
          To learn more about how we're different, you might be interested in:
        </p>
        <ul className="list-disc">
          {advancedTopicsDocs.map(({ link, description }) => (
            <li className="ml-6" key={link}>
              <DocExternalLink link={link} description={description} />
            </li>
          ))}
        </ul>
      </div>
    </div>
  );
};
