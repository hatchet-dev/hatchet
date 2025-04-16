import os

import trafilatura
from mkdocs.config.defaults import MkDocsConfig
from mkdocs.plugins import BasePlugin
from mkdocs.structure.pages import Page


class MarkdownExportPlugin(BasePlugin):  # type: ignore
    def on_post_page(
        self, output_content: str, page: Page, config: MkDocsConfig
    ) -> str:
        md_content = trafilatura.extract(
            output_content,
            url=None,
            output_format="markdown",
            include_tables=True,
            include_comments=False,
            include_links=True,
            include_formatting=True,
            include_images=False,
        )

        if not md_content:
            return output_content

        output_dir = "docs/gen"

        dest = os.path.splitext(page.file.dest_path)[0] + ".md"
        out_path = os.path.join(output_dir, dest)
        os.makedirs(os.path.dirname(out_path), exist_ok=True)

        with open(out_path, "w", encoding="utf-8") as f:
            f.write(md_content)

        return output_content
