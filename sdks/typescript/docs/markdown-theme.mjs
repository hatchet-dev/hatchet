// @ts-check
import { MarkdownTheme, MarkdownThemeContext } from 'typedoc-plugin-markdown';

/**
 * Custom TypeDoc theme for Hatchet TS SDK docs.
 *   - removes function/method signature titles from the output.
 *   - splits PascalCase class names into space-separated words for page titles.
 *   - removes type parameters table and title from the output. (generics)
 */
export function load(app) {
  app.renderer.defineTheme('hatchet-ts-docs', HatchetDocsTheme);
}

class HatchetDocsTheme extends MarkdownTheme {
  getRenderContext(page) {
    return new HatchetDocsContext(this, page, this.application.options);
  }

  render(page) {
    return removeTypeParametersTitle(super.render(page));
  }
}

/** @param {string} name */
function splitPascalCase(name) {
  return name.replace(/([a-z])([A-Z])/g, '$1 $2');
}

function removeTypeParametersTitle(content) {
  return content.replace(/#{1,6}\s+Type Parameters\n*/g, '');
}

class HatchetDocsContext extends MarkdownThemeContext {
  /** @param {ConstructorParameters<typeof MarkdownThemeContext>} args */
  constructor(...args) {
    super(...args);
    const origPageTitle = this.partials.pageTitle;
    this.partials = {
      ...this.partials,
      signatureTitle: () => '',
      typeParametersTable: () => '',
      typeParametersList: () => '',
      pageTitle: () => splitPascalCase(origPageTitle()),
    };
  }
}
