// @ts-check
import { MarkdownTheme, MarkdownThemeContext } from 'typedoc-plugin-markdown';

/**
 * Custom TypeDoc theme for Hatchet TS SDK docs.
 *   - removes function/method signature titles from the output.
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

function removeTypeParametersTitle(content) {
  return content.replace(/#{1,6}\s+Type Parameters\n*/g, '');
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
