import os
from typing import cast

from bs4 import BeautifulSoup, Tag
from markdownify import markdownify  # type: ignore[import-untyped]
from mkdocs.config.defaults import MkDocsConfig
from mkdocs.plugins import BasePlugin
from mkdocs.structure.pages import Page

from docs.generator.shared import TMP_GEN_PATH


class MarkdownExportPlugin(BasePlugin):  # type: ignore
    def __init__(self) -> None:
        super().__init__()
        self.soup: BeautifulSoup
        self.page_source_path: str

    def _remove_async_tags(self) -> "MarkdownExportPlugin":
        spans = self.soup.find_all("span", class_="doc doc-labels")

        for span in spans:
            if span.find(string="async") or (
                span.text and "async" == span.get_text().strip()
            ):
                span.decompose()

        return self

    def _remove_hash_links(self) -> "MarkdownExportPlugin":
        links = self.soup.find_all("a", class_="headerlink")
        for link in links:
            href = cast(str, link["href"])
            if href.startswith("#"):
                link.decompose()

        return self

    def _remove_toc(self) -> "MarkdownExportPlugin":
        tocs = self.soup.find_all("nav")
        for toc in tocs:
            toc.decompose()

        return self

    def _remove_footer(self) -> "MarkdownExportPlugin":
        footer = self.soup.find("footer")
        if footer and isinstance(footer, Tag):
            footer.decompose()

        return self

    def _remove_navbar(self) -> "MarkdownExportPlugin":
        navbar = self.soup.find("div", class_="navbar")
        if navbar and isinstance(navbar, Tag):
            navbar.decompose()

        navbar_header = self.soup.find("div", class_="navbar-header")
        if navbar_header and isinstance(navbar_header, Tag):
            navbar_header.decompose()
        navbar_collapse = self.soup.find("div", class_="navbar-collapse")
        if navbar_collapse and isinstance(navbar_collapse, Tag):
            navbar_collapse.decompose()

        return self

    def _remove_keyboard_shortcuts_modal(self) -> "MarkdownExportPlugin":
        modal = self.soup.find("div", id="mkdocs_keyboard_modal")

        if modal and isinstance(modal, Tag):
            modal.decompose()

        return self

    def _remove_title(self) -> "MarkdownExportPlugin":
        title = self.soup.find("h1", class_="title")

        if title and isinstance(title, Tag):
            title.decompose()

        return self

    def _remove_property_tags(self) -> "MarkdownExportPlugin":
        property_tags = self.soup.find_all("code", string="property")

        for tag in property_tags:
            tag.decompose()

        return self

    def _interpolate_docs_links(self) -> "MarkdownExportPlugin":
        links = self.soup.find_all("a")
        page_depth = self.page_source_path.count("/")

        ## Using the depth + 2 here because the links are relative to the root of
        ## the SDK docs subdir, which sits at `/sdks/python` (two levels below the root)
        dirs_up_prefix = "../" * (page_depth + 2)

        for link in links:
            href = link.get("href")

            if not href:
                continue

            href = cast(str, link["href"])

            if href.startswith("https://docs.hatchet.run/"):
                link["href"] = href.replace("https://docs.hatchet.run/", dirs_up_prefix)

        return self

    def _preprocess_html(self, content: str) -> str:
        self.soup = BeautifulSoup(content, "html.parser")

        (
            self._remove_async_tags()
            ._remove_hash_links()
            ._remove_toc()
            ._remove_footer()
            ._remove_keyboard_shortcuts_modal()
            ._remove_navbar()
            ._remove_title()
            ._remove_property_tags()
            ._interpolate_docs_links()
        )

        return str(self.soup)

    def on_post_page(
        self, output_content: str, page: Page, config: MkDocsConfig
    ) -> str:
        self.page_source_path = page.file.src_uri

        content = self._preprocess_html(output_content)
        md_content = markdownify(content, heading_style="ATX", wrap=False)

        if not md_content:
            return content

        dest = os.path.splitext(page.file.dest_path)[0] + ".md"
        out_path = os.path.join(TMP_GEN_PATH, dest)
        os.makedirs(os.path.dirname(out_path), exist_ok=True)

        with open(out_path, "w", encoding="utf-8") as f:
            f.write(md_content)

        return content
