// @ts-check
import { MarkdownTheme, MarkdownThemeContext } from 'typedoc-plugin-markdown';

/**
 * Custom TypeDoc theme for Hatchet TS SDK docs.
 *   - removes function/method signature titles from the output.
 *   - removes type parameters table and title from the output. (generics)
 *   - removes unwanted headings from the output.
 *   - spaces out PascalCase class names in headings (e.g. CronClient → Cron Client).
 */
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

class HatchetDocsTheme extends MarkdownTheme {
  getRenderContext(page) {
    return new HatchetDocsContext(this, page, this.application.options);
  }

  render(page) {
    return spaceOutHeadings(removeUnwantedHeadings(super.render(page)));
  }
}

function removeUnwantedHeadings(content) {
  return content
    .replace(/#{1,6}\s+Type Parameters\n*/g, '')
    .replace(/#{1,6}\s+Call Signature\n*/g, '')
    .replace(/#{1,6}\s+Classes\n*/g, '')
    .replace(/#{1,6}\s+client\/features\/\S+\n*/g, '');
}


function spaceOutHeadings(content) {
  return content.replace(/^(#{1,6} )(\S+)$/gm, (match, hashes, name) =>
    hashes + (HEADING_RENAMES[name] ?? name)
  );
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
