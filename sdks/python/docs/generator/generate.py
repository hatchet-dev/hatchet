import asyncio
import os

from docs.generator.llm import parse_markdown
from docs.generator.paths import crawl_directory, find_child_paths
from docs.generator.shared import TMP_GEN_PATH
from docs.generator.types import Document
from docs.generator.utils import gather_max_concurrency, rm_rf


async def clean_markdown_with_openai(document: Document) -> None:
    print("Generating mdx for", document.readable_source_path)

    with open(document.source_path, "r", encoding="utf-8") as f:
        original_md = f.read()

    content = await parse_markdown(original_markdown=original_md)

    if not content:
        return None

    with open(document.mdx_output_path, "w", encoding="utf-8") as f:
        f.write(content)


def generate_sub_meta_entry(child: str) -> str:
    child = child.replace("/", "")
    return f"""
        "{child}": {{
            "title": "{child.title()}",
            "theme": {{
                "toc": true
            }}
        }},
    """


def generate_meta_js(docs: list[Document], children: set[str]) -> str:
    prefix = docs[0].directory
    subentries = [doc.meta_js_entry for doc in docs] + [
        generate_sub_meta_entry(child.replace(prefix, "")) for child in children
    ]
    sorted_subentries = sorted(
        subentries,
        key=lambda x: (
            "aaaaaaaa"
            if "index" in (key := x.split(":")[0].strip('"').lower())
            else key
        ),
    )

    entries = "".join(sorted_subentries)

    return f"export default {{{entries}}}"


async def run() -> None:
    rm_rf(TMP_GEN_PATH)

    try:
        os.system("poetry run mkdocs build")
        documents = crawl_directory(TMP_GEN_PATH)

        await gather_max_concurrency(
            *[clean_markdown_with_openai(d) for d in documents], max_concurrency=10
        )

        meta_js_out_paths = {d.mdx_output_meta_js_path for d in documents}

        for path in meta_js_out_paths:
            relevant_documents = [
                d for d in documents if d.mdx_output_meta_js_path == path
            ]

            exemplar = relevant_documents[0]

            directory = exemplar.directory

            children = find_child_paths(directory, documents)

            meta = generate_meta_js(relevant_documents, children)

            out_path = exemplar.mdx_output_meta_js_path

            with open(out_path, "w", encoding="utf-8") as f:
                f.write(meta)


        os.chdir("../../frontend/docs")
        os.system("pnpm lint:fix")
    finally:
        # rm_rf("docs/site")
        # rm_rf("site")
        rm_rf(TMP_GEN_PATH)


def main() -> None:
    asyncio.run(run())


if __name__ == "__main__":
    main()
