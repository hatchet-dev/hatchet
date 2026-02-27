// @ts-check
import { MarkdownTheme, MarkdownThemeContext } from 'typedoc-plugin-markdown';

export function load(app) {
  app.renderer.defineTheme('hatchet-ts-docs', HatchetDocsTheme);
}

const HEADING_RENAMES = {
  CronClient: 'Cron Client',
  FiltersClient: 'Filters Client',
  MetricsClient: 'Metrics Client',
  RateLimitsClient: 'Rate Limits Client',
  RunsClient: 'Runs Client',
  SchedulesClient: 'Schedules Client',
  WorkersClient: 'Workers Client',
  WorkflowsClient: 'Workflows Client',
  WebhooksClient: 'Webhooks Client',
};

const HEADINGS_TO_REMOVE = [
  'Type Parameters',
  'Call Signature',
  'Classes',
];

const HEADINGS_TO_BOLD = [
  'Parameters',
  'Returns',
  'Throws',
  'Overrides',
  'Accessors',
  'Get Signature',
  'Implementation of',
  'Note',
  'Alias',
  'Implements',
  'Extended by'
];

class HatchetDocsTheme extends MarkdownTheme {
  getRenderContext(page) {
    return new HatchetDocsContext(this, page, this.application.options);
  }

  render(page) {
    const transforms = [removeUnwantedHeadings, stripGenericTypeParams, unescapeHeadingUnderscores, spaceOutHeadings, codeWrapMethodHeadings];
    return transforms.reduce((content, fn) => fn(content), super.render(page));
  }
}

function removeUnwantedHeadings(content) {
  let result = content.replace(/#{1,6}\s+client\/features\/\S+\n*/g, '');
  for (const heading of HEADINGS_TO_REMOVE) {
    result = result.replace(new RegExp(`#{1,6}\\s+${heading}\\n*`, 'g'), '');
  }
  for (const heading of HEADINGS_TO_BOLD) {
    result = result.replace(new RegExp(`^#{1,6}\\s+(${heading})$`, 'gmi'), '**$1**');
  }
  return result;
}

function stripGenericTypeParams(content) {
  return content.replace(/^(#{1,6} .+?)\\<[^>]*>/gm, '$1');
}

function unescapeHeadingUnderscores(content) {
  return content.replace(/^(#{1,6} .+)$/gm, (match) => match.replace(/\\_/g, '_'));
}

function spaceOutHeadings(content) {
  return content.replace(/^(#{1,6} )(\S+)$/gm, (match, hashes, name) =>
    hashes + (HEADING_RENAMES[name] ?? name)
  );
}

function codeWrapMethodHeadings(content) {
  return content
    .replace(/^(#{3,6} )(.+\(\))$/gm, (match, hashes, name) => `${hashes}\`${name}\``)
    .replace(/^(#{4,6} )([a-z]\w*)$/gm, (match, hashes, name) => `${hashes}\`${name}\``);
}

class HatchetDocsContext extends MarkdownThemeContext {
  /** @param {ConstructorParameters<typeof MarkdownThemeContext>} args */
  constructor(...args) {
    super(...args);
    this.partials = {
      ...this.partials,
      signatureTitle: () => '',
      typeParametersTable: () => '',
    };
  }
}
