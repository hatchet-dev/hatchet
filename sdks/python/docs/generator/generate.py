import argparse
import asyncio
import os
from typing import cast

from docs.generator.llm import client
from docs.generator.paths import crawl_directory, find_child_paths
from docs.generator.prompts import create_prompt_messages
from docs.generator.shared import TMP_GEN_PATH
from docs.generator.types import Document
from docs.generator.utils import gather_max_concurrency, rm_rf


async def clean_markdown_with_openai(document: Document) -> None:
    print("Generating mdx for", document.readable_source_path)

    with open(document.source_path, "r", encoding="utf-8") as f:
        original_md = f.read()

    response = await client.chat.completions.create(
        model="gpt-4o", messages=create_prompt_messages(original_md)
    )

    content = response.choices[0].message.content

    if not content:
        return None

    with open(document.mdx_output_path, "w", encoding="utf-8") as f:
        f.write(content)


def generate_sub_meta_entry(child: str) -> str:
    child = child.replace("/", "")
    return f"""
        "{child}": {{
            "title": "{child.title()}",
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


async def run(paths: list[str], include_all: bool) -> None:
    rm_rf(TMP_GEN_PATH)

    try:
        os.system("poetry run mkdocs build")
        documents = crawl_directory(TMP_GEN_PATH, include_all, paths)

        await gather_max_concurrency(
            *[clean_markdown_with_openai(d) for d in documents], max_concurrency=10
        )

        directories = {d.directory for d in documents}

        for directory in directories:
            children = find_child_paths(directory, documents)

            meta = generate_meta_js(
                [d for d in documents if d.directory == directory], children
            )
            out_path = (
                directory.replace(
                    TMP_GEN_PATH, "../../frontend/docs/pages/sdks/python"
                ).replace(".md", ".mdx")
                + "/_meta.js"
            )

            with open(out_path, "w", encoding="utf-8") as f:
                f.write(meta)

        os.chdir("../../frontend/docs")
        os.system("pnpm lint:fix")
    finally:
        # rm_rf("docs/site")
        # rm_rf("site")
        rm_rf(TMP_GEN_PATH)


def main() -> None:
    parser = argparse.ArgumentParser()
    group = parser.add_mutually_exclusive_group(required=True)
    group.add_argument("--all", action="store_true", help="Generate docs for all files")
    group.add_argument(
        "paths",
        nargs="*",
        help="Paths to specific files to generate docs for (e.g., runnables.md)",
    )

    args = parser.parse_args()

    paths = cast(list[str], args.paths)
    include_all = cast(bool, args.all)

    asyncio.run(run(paths, include_all))


if __name__ == "__main__":
    main()
