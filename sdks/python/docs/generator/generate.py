import argparse
import os
import re

from docs.generator.doc_types import Document
from docs.generator.paths import crawl_directory, find_child_paths
from docs.generator.shared import TMP_GEN_PATH
from docs.generator.utils import rm_rf

CODE_SPAN_PATTERN = re.compile(r"(`[^`]*`)")


def escape_mdx_line(line: str) -> str:
    is_table_row = line.lstrip().startswith("|")
    segments = []

    for segment in CODE_SPAN_PATTERN.split(line):
        if segment.startswith("`"):
            if is_table_row:
                segment = re.sub(r"(?<!\\)\|", r"\\|", segment)
            segments.append(segment)
        else:
            segments.append(re.sub(r"(?<!\\)([<{}])", r"\\\1", segment))

    return "".join(segments)


def to_mdx(content: str) -> str:
    first_heading = re.search(r"^#\s", content, flags=re.MULTILINE)
    if first_heading:
        content = content[first_heading.start() :]

    lines = []
    in_fence = False

    for line in content.splitlines(keepends=True):
        if line.lstrip().startswith("```"):
            if not in_fence and line.strip() == "```":
                line = line.replace("```", "```python", 1)
            in_fence = not in_fence
            lines.append(line)
        elif in_fence:
            lines.append(line)
        else:
            lines.append(escape_mdx_line(line))

    return "".join(lines)


def clean_markdown(document: Document) -> None:
    print("Generating mdx for", document.readable_source_path)

    with open(document.source_path, "r", encoding="utf-8") as f:
        original_md = f.read()

    with open(document.mdx_output_path, "w", encoding="utf-8") as f:
        f.write(to_mdx(original_md))


def generate_sub_meta_entry(child: str) -> str:
    child = child.replace("/", "")
    return f"""
        "{child}": {{
            "title": "{child.replace("-", " ").title()}",
            "theme": {{
                "toc": true
            }},
        }},
    """


def generate_meta_js(docs: list[Document], children: set[str]) -> str:
    prefix = docs[0].directory
    subentries = [doc.meta_js_entry for doc in docs] + [
        generate_sub_meta_entry(child.replace(prefix, "")) for child in children
    ]

    sorted_subentries = sorted(
        subentries,
        key=lambda x: x.strip().split(":")[0].strip('"').lower(),
    )

    entries = "".join(sorted_subentries)

    return f"export default {{{entries}}}"


def update_meta_js(documents: list[Document]) -> None:
    meta_js_out_paths = {d.mdx_output_meta_js_path for d in documents}

    for path in meta_js_out_paths:
        relevant_documents = [d for d in documents if d.mdx_output_meta_js_path == path]

        exemplar = relevant_documents[0]

        directory = exemplar.directory

        children = find_child_paths(directory, documents)

        meta = generate_meta_js(relevant_documents, children)

        out_path = exemplar.mdx_output_meta_js_path

        with open(out_path, "w", encoding="utf-8") as f:
            f.write(meta)


def run(selections: list[str]) -> None:
    rm_rf(TMP_GEN_PATH)

    try:
        os.system("poetry run mkdocs build")
        documents = crawl_directory(TMP_GEN_PATH, selections)

        for document in documents:
            clean_markdown(document)

        if not selections:
            update_meta_js(documents)

        os.chdir("../../frontend/docs")
        os.system("pnpm lint:fix")
    finally:
        rm_rf("docs/site")
        rm_rf("site")
        rm_rf(TMP_GEN_PATH)


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument(
        "--select",
        type=str,
        help="Comma-separated list of doc names to generate (e.g. client,context,runnables). Note that this will prevent the `_meta.js` file from being generated.",
    )

    args = parser.parse_args()

    selections = (
        [f"{name.strip()}.md" for name in args.select.split(",")] if args.select else []
    )

    run(selections)


if __name__ == "__main__":
    main()
